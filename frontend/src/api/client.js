import axios from "axios";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

const apiClient = axios.create({
  baseURL: API_URL,
  // ❗ Don't force Content-Type globally.
  // JSON requests will set it automatically; FormData must not be overridden.
});

// Add token to requests
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem("token");

    // Ensure headers exists (axios sometimes uses undefined)
    config.headers = config.headers || {};

    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// Handle 401 errors
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      window.location.href = "/login";
    }
    return Promise.reject(error);
  }
);

// Auth API
export const authAPI = {
  login: (email, password) => apiClient.post("/api/login", { email, password }),
  register: (userData) => apiClient.post("/api/register", userData),
  getMe: () => apiClient.get("/api/me"),
};

// Patient API
export const patientAPI = {
  /**
   * Upload DICOM file.
   * Expects FormData with:
   *  - file: File
   *  - destination: "diagnocat" | "orthanc"
   *
   * onProgress(percentInt) is optional.
   */
  uploadDICOM: (formData, onProgress) => {
    return apiClient.post("/api/patient/upload", formData, {
      // ✅ IMPORTANT: do NOT set Content-Type manually for FormData.
      // The browser will set proper multipart boundary.
      onUploadProgress: (progressEvent) => {
        if (!onProgress) return;

        const total = progressEvent.total ?? 0;
        if (!total) return;

        const percent = Math.round((progressEvent.loaded * 100) / total);
        onProgress(percent);
      },
    });
  },

  getStudies: () => apiClient.get("/api/patient/studies"),
};

export const diagnocatAPI = {
  getAnalyses: () => apiClient.get("/api/patient/diagnocat/analyses"),
  refreshAnalysis: (analysisId) =>
    apiClient.get(`/api/patient/diagnocat/analyses/${analysisId}/refresh`),
  sendStudy: (studyId) =>
    apiClient.post("/api/patient/diagnocat/send", { study_id: studyId }),
};

export default apiClient;
