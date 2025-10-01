package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var MongoURI string
var DefaultDBContextTimeout int
var GoogleClientID string
var GoogleClientSecret string

var GoogleOauthConfig oauth2.Config

func LoadConfig() {

	// load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	GoogleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	GoogleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	MongoURI = os.Getenv("MONGO_URI")
	DefaultDBContextTimeout = 10

	GoogleOauthConfig = oauth2.Config{
		RedirectURL:  "http://localhost:3000/api/v1/auth/google/callback",
		ClientID:     GoogleClientID,
		ClientSecret: GoogleClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}
