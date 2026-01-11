import axios from "axios";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

const apiClient = axios.create({
  baseURL: API_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

// Add token to requests
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem("token");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
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
  // UPDATED: Now accepts FormData with destination parameter
  uploadDICOM: (formData, onUploadProgress) =>
    apiClient.post("/api/patient/upload", formData, {
      headers: { "Content-Type": "multipart/form-data" },
      onUploadProgress: (progressEvent) => {
        const progress = Math.round(
          (progressEvent.loaded * 100) / progressEvent.total
        );
        onUploadProgress(progress);
      },
    }),

  getStudies: () => apiClient.get("/api/patient/studies"),
};

export const diagnocatAPI = {
  sendStudy: (studyId) =>
    apiClient.post("/api/patient/diagnocat/send", { study_id: studyId }),

  getAnalyses: () => apiClient.get("/api/patient/diagnocat/analyses"),

  refreshAnalysis: (analysisId) =>
    apiClient.get(`/api/patient/diagnocat/analyses/${analysisId}/refresh`),
};

export default apiClient;
