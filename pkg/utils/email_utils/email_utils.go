package email_utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	envEmailAddress    = "EMAIL_USERNAME"
	envEmailServerLink = "EMAIL_SERVER_LINK"
)

var (
	email  = "sinotic.co@gmail.com"
	server = "http://localhost:8080"
)

func init() {
	if e := os.Getenv(envEmailAddress); e != "" {
		email = e
	}
	if s := os.Getenv(envEmailServerLink); s != "" {
		server = s
	}
}

func SendEmail(ctx context.Context, recipientEmail string, user domain.User) (*domain.EmailVerification, rest_errors.RestError) {
	emailVerification := domain.EmailVerification{
		Id:             uuid.NewString(),
		CreateDateTime: time.Now().UTC(),
		UserId:         user.Id,
		Email:          user.Email,
	}

	b, err := ioutil.ReadFile("../../oauth/gmail/credentials.json")
	if err != nil {
		logger.Error("Unable to read client secret file:", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailSendScope)
	if err != nil {
		logger.Error("Unable to parse client secret file config:", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		logger.Error("Unable to retrieve Gmail client:", err)
	}

	// Create Message
	message := fmt.Sprintf("Please click following link to verify email:\n\n%s/api/v1/auth/verifyEmail?userId=%s&authId=%s", server, user.Id, emailVerification.Id)

	// Send message
	ret, sendErr := sendMail(email, recipientEmail, "Please Verify Your Email", message, srv)
	if sendErr != nil {
		logger.Error("could not send email", sendErr)
		return nil, rest_errors.NewInternalServerError("could not send email", errors.New("smtp error"))
	}
	fmt.Println("Email sent:", ret)

	logger.Info(fmt.Sprintf("email sent to %s for user id %s", recipientEmail, user.Id), context_utils.GetTraceAndClientIds(ctx)...)
	return &emailVerification, nil
}

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "../../oauth/gmail/token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authUrl := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Println("Go to the following link and type the autorization code:\n", authUrl)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		logger.Error("Unable to read auth code:", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		logger.Error("Unable to retrieve token:", err)
	}

	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.Error("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func sendMail(from string, to string, title string, message string, srv *gmail.Service) (bool, error) {
	// Create the message
	msgStr := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, title, message)
	msg := []byte(msgStr)
	// Get raw
	gMessage := &gmail.Message{Raw: base64.URLEncoding.EncodeToString(msg)}

	// Send the message
	_, err := srv.Users.Messages.Send("me", gMessage).Do()
	if err != nil {
		fmt.Println("Could not send mail>", err)
		return false, err
	}
	return true, nil
}
