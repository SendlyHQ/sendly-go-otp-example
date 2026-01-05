package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/SendlyHQ/sendly-go/sendly"
	"github.com/joho/godotenv"
)

//go:embed templates/*
var templates embed.FS

type SendOTPRequest struct {
	Phone string `json:"phone"`
}

type SendOTPResponse struct {
	VerificationID string `json:"verificationId"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
}

type VerifyOTPRequest struct {
	VerificationID string `json:"verificationId"`
	Code           string `json:"code"`
}

type VerifyOTPResponse struct {
	Success  bool   `json:"success"`
	Status   string `json:"status,omitempty"`
	Error    string `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
}

var client *sendly.Client

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	apiKey := os.Getenv("SENDLY_API_KEY")
	if apiKey == "" {
		log.Fatal("SENDLY_API_KEY environment variable is required")
	}

	client = sendly.NewClient(apiKey)

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/send-otp", sendOTP)
	http.HandleFunc("/verify", serveVerify)
	http.HandleFunc("/verify-otp", verifyOTP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	content, err := templates.ReadFile("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(content)
}

func serveVerify(w http.ResponseWriter, r *http.Request) {
	content, err := templates.ReadFile("templates/verify.html")
	if err != nil {
		http.Error(w, "Error loading page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(content)
}

func sendOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, SendOTPResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Phone == "" {
		respondJSON(w, http.StatusBadRequest, SendOTPResponse{
			Success: false,
			Error:   "Phone number is required",
		})
		return
	}

	ctx := context.Background()
	verification, err := client.Verify.Send(ctx, &sendly.SendVerificationRequest{
		To: req.Phone,
	})

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, SendOTPResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to send OTP: %v", err),
		})
		return
	}

	respondJSON(w, http.StatusOK, SendOTPResponse{
		Success:        true,
		VerificationID: verification.ID,
	})
}

func verifyOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, VerifyOTPResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.VerificationID == "" || req.Code == "" {
		respondJSON(w, http.StatusBadRequest, VerifyOTPResponse{
			Success: false,
			Error:   "Verification ID and code are required",
		})
		return
	}

	ctx := context.Background()
	result, err := client.Verify.Check(ctx, req.VerificationID, &sendly.CheckVerificationRequest{
		Code: req.Code,
	})

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, VerifyOTPResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to verify OTP: %v", err),
		})
		return
	}

	if result.Status == "verified" {
		respondJSON(w, http.StatusOK, VerifyOTPResponse{
			Success: true,
			Status:  result.Status,
			Message: "Phone number verified successfully!",
		})
	} else {
		respondJSON(w, http.StatusOK, VerifyOTPResponse{
			Success: false,
			Status:  result.Status,
			Error:   "Invalid verification code",
		})
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
