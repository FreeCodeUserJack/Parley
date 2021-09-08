package sms_utils

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	envAccountSid = "TWILIO_ACCOUNT_SID"
	envAuthToken  = "TWILIO_AUTH_TOKEN"
)

var (
	accountSid = "ACaecec1be75d98f1b427bd85d4d769ab8"
	authToken  = "3984c3b9ee3739743d7b50e537fffb0d"
)

func init() {
	if accsid := os.Getenv(envAccountSid); accsid != "" {
		accountSid = accsid
	}

	if authtok := os.Getenv(envAuthToken); authtok != "" {
		authToken = authtok
	}
}

func SendSMS(ctx context.Context, phone, otp string) {
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"

	msgData := url.Values{}
	msgData.Set("To", "+1"+phone)
	msgData.Set("From", "+12565108104")

	if otp == "" {
		otp, _ = GenerateOTP(6)
	}
	msgData.Set("Body", "Please login and enter your code: "+otp)

	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &msgDataReader)
	req.SetBasicAuth(accountSid, authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := client.Do(req)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var data map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&data)
		if err == nil {
			fmt.Println(data["sid"])
		}
	} else {
		fmt.Println(resp.Status)
	}
}

const otpChars = "1234567890"

func GenerateOTP(length int) (string, error) {
	buffer := make([]byte, length)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}

	otpCharsLength := len(otpChars)
	for i := 0; i < length; i++ {
		buffer[i] = otpChars[int(buffer[i])%otpCharsLength]
	}

	return string(buffer), nil
}
