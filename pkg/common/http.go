package common

import (
	"encoding/json"
	"log"
	"net/http"
)

// HTTPResponseWriter provides utilities for writing HTTP responses
type HTTPResponseWriter struct{}

// NewHTTPResponseWriter creates a new HTTP response writer
func NewHTTPResponseWriter() *HTTPResponseWriter {
	return &HTTPResponseWriter{}
}

// WriteJSONResponse writes a JSON response with the specified status code
func (w *HTTPResponseWriter) WriteJSONResponse(httpWriter http.ResponseWriter, data interface{}, statusCode int) {
	httpWriter.Header().Set("Content-Type", "application/json")
	httpWriter.WriteHeader(statusCode)
	
	if err := json.NewEncoder(httpWriter).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(httpWriter, "Error encoding response", http.StatusInternalServerError)
	}
}

// WriteErrorResponse writes a standard error response
func (w *HTTPResponseWriter) WriteErrorResponse(httpWriter http.ResponseWriter, message string, statusCode int) {
	errorResponse := &ErrorResponse{
		Error: message,
		Code:  statusCode,
	}
	w.WriteJSONResponse(httpWriter, errorResponse, statusCode)
}

// WriteSuccessResponse writes a standard success response
func (w *HTTPResponseWriter) WriteSuccessResponse(httpWriter http.ResponseWriter, data interface{}, message string) {
	response := &Response{
		Success: true,
		Message: message,
		Data:    data,
	}
	w.WriteJSONResponse(httpWriter, response, http.StatusOK)
}

// WriteUnauthorized writes an unauthorized response
func (w *HTTPResponseWriter) WriteUnauthorized(httpWriter http.ResponseWriter, message string) {
	w.WriteErrorResponse(httpWriter, message, http.StatusUnauthorized)
}

// WriteForbidden writes a forbidden response
func (w *HTTPResponseWriter) WriteForbidden(httpWriter http.ResponseWriter, message string) {
	w.WriteErrorResponse(httpWriter, message, http.StatusForbidden)
}

// WriteBadRequest writes a bad request response
func (w *HTTPResponseWriter) WriteBadRequest(httpWriter http.ResponseWriter, message string) {
	w.WriteErrorResponse(httpWriter, message, http.StatusBadRequest)
}

// WriteInternalServerError writes an internal server error response
func (w *HTTPResponseWriter) WriteInternalServerError(httpWriter http.ResponseWriter, message string) {
	w.WriteErrorResponse(httpWriter, message, http.StatusInternalServerError)
}

// WriteMethodNotAllowed writes a method not allowed response
func (w *HTTPResponseWriter) WriteMethodNotAllowed(httpWriter http.ResponseWriter) {
	w.WriteErrorResponse(httpWriter, "Method not allowed", http.StatusMethodNotAllowed)
}

// WriteHealthResponse writes a health check response
func (w *HTTPResponseWriter) WriteHealthResponse(httpWriter http.ResponseWriter, status interface{}) {
	w.WriteJSONResponse(httpWriter, status, http.StatusOK)
}

// Global instance for backward compatibility
var GlobalHTTPWriter = NewHTTPResponseWriter()

// Package-level convenience functions
func WriteJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	GlobalHTTPWriter.WriteJSONResponse(w, data, statusCode)
}

func WriteErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	GlobalHTTPWriter.WriteErrorResponse(w, message, statusCode)
}

func WriteSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	GlobalHTTPWriter.WriteSuccessResponse(w, data, message)
}

func WriteUnauthorized(w http.ResponseWriter, message string) {
	GlobalHTTPWriter.WriteUnauthorized(w, message)
}

func WriteForbidden(w http.ResponseWriter, message string) {
	GlobalHTTPWriter.WriteForbidden(w, message)
}

func WriteBadRequest(w http.ResponseWriter, message string) {
	GlobalHTTPWriter.WriteBadRequest(w, message)
}

func WriteInternalServerError(w http.ResponseWriter, message string) {
	GlobalHTTPWriter.WriteInternalServerError(w, message)
}

func WriteMethodNotAllowed(w http.ResponseWriter) {
	GlobalHTTPWriter.WriteMethodNotAllowed(w)
}