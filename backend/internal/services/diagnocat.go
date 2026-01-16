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

	"github.com/joho/godotenv"
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

// These are not currently used in the working flow, but kept for compatibility.
type UploadSessionRequest struct {
	PatientUID string `json:"patient_uid"`
	StudyUID   string `json:"study_uid,omitempty"`
	StudyType  string `json:"study_type"`
}

// NOTE: /v1/upload/open-session in your working code supports ONLY {"study_uid": "..."}.
// Keep this struct if you want, but we use a map[string]string for correctness.
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
	Error      json.RawMessage `json:"error,omitempty"`
	Diagnoses  map[string]any  `json:"diagnoses,omitempty"`
}

type StudyCreateRequest struct {
	StudyName string `json:"study_name,omitempty"`
	StudyType string `json:"study_type"`           // "CBCT", "PANORAMA", "FMX", "STL"
	StudyDate string `json:"study_date,omitempty"` // e.g. "2026-01-11"
}

type CreatePatientRequest struct {
	NamePart1   string `json:"name_part1"`
	NamePart2   string `json:"name_part2"`
	Gender      string `json:"gender,omitempty"`
	DateOfBirth string `json:"date_of_birth,omitempty"`
	PatientID   string `json:"patient_id,omitempty"`
}

type PatientResponse struct {
	UID         string `json:"uid"`
	NamePart1   string `json:"name_part1"`
	NamePart2   string `json:"name_part2"`
	Gender      string `json:"gender,omitempty"`
	DateOfBirth string `json:"date_of_birth,omitempty"`
	PatientID   string `json:"patient_id,omitempty"`
}

type UploadStudyResult struct {
	PatientUID   string `json:"patient_uid"`
	StudyUID     string `json:"study_uid"`
	StudyIDV3    string `json:"study_id_v3"`
	SessionID    string `json:"session_id"`
	AnalysisUID  string `json:"analysis_uid"`
	AnalysisIDV3 string `json:"analysis_id_v3"`
	Status       string `json:"status"`
}

type StudyResponse struct {
	UID  string `json:"uid"`
	IDV3 string `json:"id_v3"`
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
		Attributes        json.RawMessage `json:"attributes"`
		PeriodontalStatus json.RawMessage `json:"periodontal_status"`
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
	report, err := s.GetAnalysisStatus(reportID)
	if err != nil {
		return nil, err
	}

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

	if dir := filepath.Dir(outPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create output dir: %w", err)
		}
	}

	req, err := http.NewRequest("GET", s.baseURL+"/v2/analyses/"+reportID+"/pdf", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

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

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = f.Close() }()

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

	// Test connection (optional)
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
	if s.userToken != "" && time.Now().Before(s.tokenExpires) {
		token := s.userToken
		s.mu.RUnlock()
		return token, nil
	}
	s.mu.RUnlock()

	if s.email == "" || s.password == "" {
		return "", fmt.Errorf("email and password not configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// double-check after acquiring lock
	if s.userToken != "" && time.Now().Before(s.tokenExpires) {
		return s.userToken, nil
	}

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
	if authResp.Token == "" {
		return "", fmt.Errorf("auth token missing in response")
	}

	s.userToken = authResp.Token
	s.tokenExpires = time.Now().Add(23 * time.Hour)

	fmt.Println("✅ Diagnocat authentication successful!")
	return s.userToken, nil
}

func (s *DiagnocatService) getHeaders() (map[string]string, error) {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	if s.apiKey != "" {
		headers["Authorization"] = "Bearer " + s.apiKey
		return headers, nil
	}

	token, err := s.getUserToken()
	if err != nil {
		return nil, err
	}
	headers["Authorization"] = "Bearer " + token
	return headers, nil
}

// UploadStudy uploads a file for an existing Diagnocat patient UID (patientID == patientUID in Diagnocat).
func (s *DiagnocatService) UploadStudy(patientID, filePath string) (*AnalysisResponse, error) {
	headers, err := s.getHeaders()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth headers: %w", err)
	}

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

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
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

	pr := &progressReader{r: f, total: size, lastLog: time.Now()}

	uploadReq, err := http.NewRequest(http.MethodPut, uploadURL, pr)
	if err != nil {
		return nil, err
	}
	uploadReq.ContentLength = size
	uploadReq.Header.Set("Content-Type", "application/octet-stream")

	uploadClient := &http.Client{Timeout: 0}
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
	fmt.Println("✅ Session closing started")

	fmt.Println("⏳ Step 5: Waiting for session processing...")

	for i := 0; i < 180; i++ {
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
			i = 999999
		case "error":
			return nil, fmt.Errorf("session processing failed: %s", info.SessionInfo.Error)
		}
	}

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

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request analysis failed: %s: %s", resp.Status, string(b))
	}

	var analysisResp AnalysisResponse
	if err := json.NewDecoder(resp.Body).Decode(&analysisResp); err != nil {
		return nil, fmt.Errorf("failed to decode analysis response: %w", err)
	}

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

func (s *DiagnocatService) UploadStudyDetailed(patientUID, filePath string) (*UploadStudyResult, error) {
	headers, err := s.getHeaders()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth headers: %w", err)
	}

	// STEP 0: Create study
	studyReq := StudyCreateRequest{
		StudyName: "Upload from app",
		StudyType: "CBCT",
		StudyDate: time.Now().UTC().Format("2006-01-02"),
	}

	studyBody, _ := json.Marshal(studyReq)
	req, _ := http.NewRequest("POST", s.baseURL+"/v2/patients/"+patientUID+"/studies", bytes.NewReader(studyBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create study request failed: %w", err)
	}
	defer resp.Body.Close()

	studyRespBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("create study failed: status %d: %s", resp.StatusCode, string(studyRespBody))
	}

	var study StudyResponse
	if err := json.Unmarshal(studyRespBody, &study); err != nil {
		return nil, fmt.Errorf("failed to decode study response: %w", err)
	}
	if study.UID == "" {
		return nil, fmt.Errorf("create study returned empty UID: %s", string(studyRespBody))
	}

	// STEP 1: Open upload session (ONLY study_uid supported)
	openReq := map[string]string{"study_uid": study.UID}
	openBody, _ := json.Marshal(openReq)

	req, _ = http.NewRequest("POST", s.baseURL+"/v1/upload/open-session", bytes.NewReader(openBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open session request failed: %w", err)
	}
	defer resp.Body.Close()

	openRespBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("open session failed: status %d: %s", resp.StatusCode, string(openRespBody))
	}

	var sessionResp UploadSessionResponse
	if err := json.Unmarshal(openRespBody, &sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode open session response: %w", err)
	}
	if sessionResp.SessionID == "" {
		return nil, fmt.Errorf("open session returned empty session_id: %s", string(openRespBody))
	}
	sessionID := sessionResp.SessionID

	// STEP 2: Request upload URL
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
		return nil, fmt.Errorf("request upload urls failed: %w", err)
	}
	defer resp.Body.Close()

	urlRespBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("request upload urls failed: status %d: %s", resp.StatusCode, string(urlRespBody))
	}

	var uploadURLsResp RequestUploadURLsResponse
	if err := json.Unmarshal(urlRespBody, &uploadURLsResp); err != nil {
		return nil, fmt.Errorf("failed to decode upload urls response: %w", err)
	}
	if !uploadURLsResp.OK || len(uploadURLsResp.UploadURLs) == 0 {
		return nil, fmt.Errorf("no upload urls returned: %s", string(urlRespBody))
	}
	uploadURL := uploadURLsResp.UploadURLs[0].URL

	// STEP 3: Upload file
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

	pr := &progressReader{r: f, total: size, lastLog: time.Now()}

	putReq, _ := http.NewRequest("PUT", uploadURL, pr)
	putReq.Header.Set("Content-Type", "application/octet-stream")
	putReq.ContentLength = size

	uploadClient := &http.Client{Timeout: 0}
	putResp, err := uploadClient.Do(putReq)
	if err != nil {
		return nil, fmt.Errorf("upload PUT failed: %w", err)
	}
	putResp.Body.Close()

	if putResp.StatusCode < 200 || putResp.StatusCode > 299 {
		return nil, fmt.Errorf("upload PUT failed: status %d", putResp.StatusCode)
	}

	// STEP 4: Close session (start close)
	closeReq := CloseSessionRequest{SessionID: sessionID}
	closeBody, _ := json.Marshal(closeReq)

	req, _ = http.NewRequest("POST", s.baseURL+"/v1/upload/start-session-close", bytes.NewReader(closeBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("close session failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("close session failed: %s: %s", resp.Status, string(b))
	}

	// STEP 5: Wait for session closed
	for i := 0; i < 180; i++ {
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
			i = 999999
		case "error":
			return nil, fmt.Errorf("session processing failed: %s", info.SessionInfo.Error)
		}
	}

	// STEP 6: Request analysis (same endpoint as UploadStudy)
	analysisReq := RequestAnalysisRequest{AnalysisType: "GP"}
	analysisBody, _ := json.Marshal(analysisReq)

	req, _ = http.NewRequest("POST", s.baseURL+"/v2/studies/"+study.UID+"/analyses", bytes.NewReader(analysisBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request analysis failed: %w", err)
	}
	defer resp.Body.Close()

	analysisRespBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("request analysis failed: status %d: %s", resp.StatusCode, string(analysisRespBody))
	}

	var analysis AnalysisResponse
	if err := json.Unmarshal(analysisRespBody, &analysis); err != nil {
		return nil, fmt.Errorf("failed to decode analysis response: %w", err)
	}

	return &UploadStudyResult{
		PatientUID:   patientUID,
		StudyUID:     study.UID,
		StudyIDV3:    study.IDV3,
		SessionID:    sessionID,
		AnalysisUID:  analysis.UID,
		AnalysisIDV3: analysis.IDV3,
		Status:       analysis.Status,
	}, nil
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

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GetAnalysisStatus failed: %s: %s", resp.Status, string(b))
	}

	var report ReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, err
	}

	if report.Status == "complete" {
		diagReq, _ := http.NewRequest("GET", s.baseURL+"/v2/analyses/"+reportID+"/diagnoses", nil)
		for k, v := range headers {
			diagReq.Header.Set(k, v)
		}

		diagResp, err := s.httpClient.Do(diagReq)
		if err == nil && diagResp.StatusCode == 200 {
			_ = json.NewDecoder(diagResp.Body).Decode(&report.Diagnoses)
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

func (s *DiagnocatService) CreatePatient(reqBody CreatePatientRequest) (*PatientResponse, error) {
	headers, err := s.getHeaders()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth headers: %w", err)
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/v2/patients", bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create patient request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("create patient failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var patient PatientResponse
	if err := json.Unmarshal(respBody, &patient); err != nil {
		return nil, fmt.Errorf("failed to decode patient response: %w", err)
	}
	if patient.UID == "" {
		return nil, fmt.Errorf("create patient returned empty UID: %s", string(respBody))
	}

	return &patient, nil
}
