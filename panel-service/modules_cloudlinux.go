package main

import "net/http"

func (s *service) handleCloudLinuxStatus(w http.ResponseWriter) {
	status := detectCloudLinuxStatus()
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: status})
}

func (s *service) handlePlatformCapabilities(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data:   buildPlatformCapabilities(),
	})
}
