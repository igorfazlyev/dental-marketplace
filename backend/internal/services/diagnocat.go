package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/joho/godotenv" // ← ADD THIS
)

type DiagnocatService struct {
	baseURL      string
	apiKey       string
	email        string
	password     string
	userToken    string
	tokenExpires time.Time
	mu           sync.RWMutex
	httpClient   *http.Client
}

type AuthTokenRequest struct {
	ClientHostID string `json:"client_host_id"`
	Email        string `json:"email"`
	Password     string `json:"password"`
}

type AuthTokenResponse struct {
	Token string `json:"token"`
}

type UploadSessionRequest struct {
	PatientUID string `json:"patient_uid"`
	StudyUID   string `json:"study_uid,omitempty"` // ← Add omitempty
	StudyType  string `json:"study_type"`
}

type OpenUploadSessionRequest struct {
	PatientUID string `json:"patient_uid"`
	StudyType  string `json:"study_type"`
}

type UploadSessionResponse struct {
	SessionID string `json:"session_id"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
}

type RequestUploadURLsRequest struct {
	SessionID string   `json:"session_id"`
	Keys      []string `json:"keys"`
}

type UploadURL struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type RequestUploadURLsResponse struct {
	OK         bool        `json:"ok"`
	Error      string      `json:"error,omitempty"`
	UploadURLs []UploadURL `json:"upload_urls"`
}

type CloseSessionRequest struct {
	SessionID string `json:"session_id"`
}

type SessionInfoResponse struct {
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
	SessionInfo struct {
		Status string `json:"status"` // "started", "closing", "closed", "error", ...
		Error  string `json:"error,omitempty"`
	} `json:"session_info"`
}

type RequestAnalysisRequest struct {
	AnalysisType string `json:"analysis_type"` // "GP", "CBCT_ORTHO", ...
}

type AnalysisResponse struct {
	UID    string `json:"uid,omitempty"`
	IDV3   string `json:"id_v3,omitempty"`
	Status string `json:"status,omitempty"`
}

type ReportResponse struct {
	ID         string          `json:"id"`
	Status     string          `json:"status"`
	Complete   bool            `json:"complete"`
	PDFUrl     string          `json:"pdf_url,omitempty"`
	WebpageUrl string          `json:"webpage_url,omitempty"`
	PreviewUrl string          `json:"preview_url,omitempty"`
	Error      json.RawMessage `json:"error,omitempty"` // ✅
	Diagnoses  map[string]any  `json:"diagnoses,omitempty"`
}

type StudyCreateRequest struct {
	StudyName string `json:"study_name,omitempty"`
	StudyType string `json:"study_type"`           // "CBCT", "PANORAMA", "FMX", "STL"
	StudyDate string `json:"study_date,omitempty"` // e.g. "2026-01-11"
}

type StudyResponse struct {
	UID  string `json:"uid"`   // legacy study uid (this is what /v1/upload/open-session wants)
	IDV3 string `json:"id_v3"` // new xid format (nice to keep)
}

type progressReader struct {
	r       io.Reader
	total   int64
	read    int64
	lastLog time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.read += int64(n)

	// log every ~2 seconds
	now := time.Now()
	if now.Sub(p.lastLog) >= 2*time.Second {
		p.lastLog = now
		percent := float64(p.read) / float64(p.total) * 100
		mbRead := float64(p.read) / 1024 / 1024
		mbTot := float64(p.total) / 1024 / 1024
		fmt.Printf("   ⏫ uploaded %.1f / %.1f MB (%.1f%%)\n", mbRead, mbTot, percent)
	}

	return n, err
}

type DiagnosesResponse struct {
	Diagnoses []struct {
		ToothNumber       int             `json:"tooth_number"`
		TextComment       string          `json:"text_comment"`
		Attributes        json.RawMessage `json:"attributes"`         // keep raw to avoid future schema changes
		PeriodontalStatus json.RawMessage `json:"periodontal_status"` // keep raw
	} `json:"diagnoses"`
}

type ReportExport struct {
	FetchedAt time.Time          `json:"fetched_at"`
	Source    string             `json:"source"`
	ReportID  string             `json:"report_id"`
	Report    ReportResponse     `json:"report"`
	Diagnoses *DiagnosesResponse `json:"diagnoses,omitempty"`
}

func (s *DiagnocatService) ExportReport(reportID string) (*ReportExport, error) {
	// 1) Get main report (includes status/complete/etc)
	report, err := s.GetAnalysisStatus(reportID)
	if err != nil {
		return nil, err
	}

	// 2) If complete, fetch diagnoses in structured format
	var diagnoses *DiagnosesResponse
	if report.Complete || report.Status == "complete" {
		headers, err := s.getHeaders()
		if err != nil {
			return nil, err
		}

		req, _ := http.NewRequest("GET", s.baseURL+"/v2/analyses/"+reportID+"/diagnoses", nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("diagnoses request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("diagnoses failed: %s: %s", resp.Status, string(b))
		}

		var d DiagnosesResponse
		if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
			return nil, fmt.Errorf("failed to decode diagnoses: %w", err)
		}
		diagnoses = &d
	}

	out := &ReportExport{
		FetchedAt: time.Now().UTC(),
		Source:    s.baseURL,
		ReportID:  reportID,
		Report:    *report,
		Diagnoses: diagnoses,
	}

	return out, nil
}

// DownloadReportPDF downloads /v2/analyses/{reportID}/pdf and writes it to outPath.
// It streams the response to disk (no huge memory usage).
func (s *DiagnocatService) DownloadReportPDF(reportID, outPath string) error {
	if reportID == "" {
		return fmt.Errorf("reportID is required")
	}
	if outPath == "" {
		return fmt.Errorf("outPath is required")
	}

	headers, err := s.getHeaders()
	if err != nil {
		return fmt.Errorf("failed to get auth headers: %w", err)
	}

	// Ensure output directory exists
	if dir := filepath.Dir(outPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create output dir: %w", err)
		}
	}

	req, err := http.NewRequest("GET", s.baseURL+"/v2/analyses/"+reportID+"/pdf", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// For PDF downloads, DO NOT force Content-Type: application/json.
	// We'll only set Authorization header.
	if auth, ok := headers["Authorization"]; ok && auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Accept", "application/pdf")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pdf request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return fmt.Errorf("pdf download failed: %s: %s", resp.Status, string(b))
	}

	// Create/overwrite file
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	// Stream copy
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to write pdf: %w", err)
	}

	return nil
}

func NewDiagnocatService() *DiagnocatService {
	_ = godotenv.Load()
	service := &DiagnocatService{
		baseURL:    getEnvOrDefault("DIAGNOCAT_API_URL", "https://app2.diagnocat.ru/partner-api"),
		apiKey:     os.Getenv("DIAGNOCAT_API_KEY"),
		email:      os.Getenv("DIAGNOCAT_EMAIL"),
		password:   os.Getenv("DIAGNOCAT_PASSWORD"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	fmt.Println("email", service.email)

	// Test connection
	service.testConnection()

	return service
}

func (s *DiagnocatService) testConnection() {
	headers, err := s.getHeaders()
	if err != nil {
		fmt.Println("⚠️ No Diagnocat credentials configured!")
		return
	}

	req, _ := http.NewRequest("GET", s.baseURL+"/v2/participants", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Failed to connect to Diagnocat API: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Println("✅ Diagnocat API connection successful!")
	} else {
		fmt.Printf("⚠️ Diagnocat API test returned status %d\n", resp.StatusCode)
	}
}

func (s *DiagnocatService) getUserToken() (string, error) {
	s.mu.RLock()
	// Check if we have a valid token
	if s.userToken != "" && time.Now().Before(s.tokenExpires) {
		token := s.userToken
		s.mu.RUnlock()
		return token, nil
	}
	s.mu.RUnlock()

	// Need to authenticate
	if s.email == "" || s.password == "" {
		return "", fmt.Errorf("email and password not configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Printf("🔐 Authenticating with Diagnocat as %s...\n", s.email)

	authReq := AuthTokenRequest{
		ClientHostID: "dental-clinic-backend",
		Email:        s.email,
		Password:     s.password,
	}

	body, _ := json.Marshal(authReq)
	req, _ := http.NewRequest("POST", s.baseURL+"/v2/auth/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var authResp AuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}

	s.userToken = authResp.Token
	// Tokens typically expire in 24 hours, set to 23 hours to be safe
	s.tokenExpires = time.Now().Add(23 * time.Hour)

	fmt.Println("✅ Diagnocat authentication successful!")
	return s.userToken, nil
}

func (s *DiagnocatService) getHeaders() (map[string]string, error) {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Prefer API key if available
	if s.apiKey != "" {
		headers["Authorization"] = "Bearer " + s.apiKey
		return headers, nil
	}

	// Fall back to user token
	token, err := s.getUserToken()
	if err != nil {
		return nil, err
	}

	headers["Authorization"] = "Bearer " + token
	return headers, nil
}

func (s *DiagnocatService) UploadStudy(patientID, filePath string) (*AnalysisResponse, error) {
	headers, err := s.getHeaders()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth headers: %w", err)
	}

	// STEP 0: Create study for patient
	fmt.Printf("📤 Step 0: Creating study for patient %s...\n", patientID)

	studyReq := StudyCreateRequest{
		StudyName: "Upload from API",
		StudyType: "CBCT",
		StudyDate: time.Now().UTC().Format("2006-01-02"),
	}

	studyBody, _ := json.Marshal(studyReq)
	req, _ := http.NewRequest("POST", s.baseURL+"/v2/patients/"+patientID+"/studies", bytes.NewReader(studyBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create study: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create study failed: %s: %s", resp.Status, string(b))
	}

	var study StudyResponse
	if err := json.NewDecoder(resp.Body).Decode(&study); err != nil {
		return nil, fmt.Errorf("failed to decode study response: %w", err)
	}
	if study.UID == "" {
		return nil, fmt.Errorf("study uid missing in response")
	}

	fmt.Printf("✅ Study created. study_uid=%s\n", study.UID)

	// STEP 1: Open upload session (ONLY study_uid is supported)
	fmt.Printf("📤 Step 1: Opening upload session for study %s...\n", study.UID)

	openReq := map[string]string{"study_uid": study.UID}
	openBody, _ := json.Marshal(openReq)

	req, _ = http.NewRequest("POST", s.baseURL+"/v1/upload/open-session", bytes.NewReader(openBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to open session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("open session failed: %s: %s", resp.Status, string(b))
	}

	var sessionResp UploadSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode session response: %w", err)
	}
	if sessionResp.SessionID == "" {
		return nil, fmt.Errorf("open-session returned empty session_id (error=%s)", sessionResp.Error)
	}
	sessionID := sessionResp.SessionID
	fmt.Printf("✅ Session opened: %s\n", sessionID)

	// STEP 2: Request upload URL(s)
	fmt.Println("📤 Step 2: Requesting upload URL...")

	key := filepath.Base(filePath)
	urlReq := RequestUploadURLsRequest{
		SessionID: sessionID,
		Keys:      []string{key},
	}
	urlBody, _ := json.Marshal(urlReq)

	req, _ = http.NewRequest("POST", s.baseURL+"/v1/upload/request-upload-urls", bytes.NewReader(urlBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request upload URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request-upload-urls failed: %s: %s", resp.Status, string(b))
	}

	var urlResp RequestUploadURLsResponse
	if err := json.NewDecoder(resp.Body).Decode(&urlResp); err != nil {
		return nil, fmt.Errorf("failed to decode URL response: %w", err)
	}
	if len(urlResp.UploadURLs) == 0 {
		return nil, fmt.Errorf("no upload_urls returned (error=%s)", urlResp.Error)
	}

	uploadURL := urlResp.UploadURLs[0].URL
	fmt.Println("✅ Got upload URL")

	// STEP 3: Upload file (streaming + content length + progress)
	fmt.Println("📤 Step 3: Uploading file to storage (streaming)...")

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	size := st.Size()

	pr := &progressReader{
		r:       f,
		total:   size,
		lastLog: time.Now(),
	}

	uploadReq, err := http.NewRequest(http.MethodPut, uploadURL, pr)
	if err != nil {
		return nil, err
	}

	// ✅ Critical for many presigned URLs:
	uploadReq.ContentLength = size

	// Content-Type usually doesn't matter for presigned PUT unless they signed it,
	// but this is safe:
	uploadReq.Header.Set("Content-Type", "application/octet-stream")

	uploadClient := &http.Client{Timeout: 0} // no timeout; large upload may take long
	resp, err = uploadClient.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("file upload failed: %s: %s", resp.Status, string(b))
	}

	fmt.Println("✅ File uploaded successfully")

	// STEP 4: Close session
	fmt.Println("📤 Step 4: Closing upload session...")

	closeReq := CloseSessionRequest{SessionID: sessionID}
	closeBody, _ := json.Marshal(closeReq)

	req, _ = http.NewRequest("POST", s.baseURL+"/v1/upload/start-session-close", bytes.NewReader(closeBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to close session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("close session failed: %s: %s", resp.Status, string(b))
	}
	fmt.Println("✅ Session closed")

	// STEP 5: Wait for session to become closed
	fmt.Println("⏳ Step 5: Waiting for session processing...")

	for i := 0; i < 180; i++ { // up to ~6 minutes @ 2s
		time.Sleep(2 * time.Second)

		req, _ = http.NewRequest("GET", s.baseURL+"/v1/upload/session-info?session_id="+sessionID, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err = s.httpClient.Do(req)
		if err != nil {
			continue
		}

		var info SessionInfoResponse
		_ = json.NewDecoder(resp.Body).Decode(&info)
		resp.Body.Close()

		switch info.SessionInfo.Status {
		case "closed":
			fmt.Println("✅ Session processing complete!")
			i = 999999 // break outer loop
		case "error":
			return nil, fmt.Errorf("session processing failed: %s", info.SessionInfo.Error)
		}
	}

	// STEP 6: Request analysis
	fmt.Println("📤 Step 6: Requesting AI analysis...")

	analysisReq := RequestAnalysisRequest{AnalysisType: "GP"}
	analysisBody, _ := json.Marshal(analysisReq)

	req, _ = http.NewRequest("POST", s.baseURL+"/v2/studies/"+study.UID+"/analyses", bytes.NewReader(analysisBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request analysis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request analysis failed: %s: %s", resp.Status, string(b))
	}

	var analysisResp AnalysisResponse
	if err := json.NewDecoder(resp.Body).Decode(&analysisResp); err != nil {
		return nil, fmt.Errorf("failed to decode analysis response: %w", err)
	}

	// Choose an ID to show/use later
	reportID := analysisResp.UID
	if reportID == "" {
		reportID = analysisResp.IDV3
	}

	fmt.Println("✅ Analysis requested!")
	fmt.Printf("   uid:   %s\n", analysisResp.UID)
	fmt.Printf("   id_v3: %s\n", analysisResp.IDV3)
	fmt.Printf("   ✅ Use this report id for status checks: %s\n", reportID)

	return &analysisResp, nil

}

func (s *DiagnocatService) GetAnalysisStatus(reportID string) (*ReportResponse, error) {
	headers, err := s.getHeaders()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", s.baseURL+"/v2/analyses/"+reportID, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// ✅ INSERT IT HERE (before json.NewDecoder)
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GetAnalysisStatus failed: %s: %s", resp.Status, string(b))
	}

	var report ReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, err
	}

	// If complete, get diagnoses too
	if report.Status == "complete" {
		diagReq, _ := http.NewRequest("GET", s.baseURL+"/v2/analyses/"+reportID+"/diagnoses", nil)
		for k, v := range headers {
			diagReq.Header.Set(k, v)
		}

		diagResp, err := s.httpClient.Do(diagReq)
		if err == nil && diagResp.StatusCode == 200 {
			json.NewDecoder(diagResp.Body).Decode(&report.Diagnoses)
			diagResp.Body.Close()
		}
	}

	return &report, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetHeaders returns authentication headers (needed for patient creation)
func (s *DiagnocatService) GetHeaders() (map[string]string, error) {
	return s.getHeaders()
}
