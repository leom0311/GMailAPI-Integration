package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
        // The file token.json stores the user's access and refresh tokens, and is
        // created automatically when the authorization flow completes for the first
        // time.
        tokFile := "token.json"
        tok, err := tokenFromFile(tokFile)
        if err != nil {
                tok = getTokenFromWeb(config)
                saveToken(tokFile, tok)
        }
        return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
        authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
        fmt.Printf("Go to the following link in your browser then type the "+
                "authorization code: \n%v\n", authURL)

        var authCode string
        if _, err := fmt.Scan(&authCode); err != nil {
                log.Fatalf("Unable to read authorization code: %v", err)
        }

        tok, err := config.Exchange(context.TODO(), authCode)
        if err != nil {
                log.Fatalf("Unable to retrieve token from web: %v", err)
        }
        return tok
}

// Retrieves a token from a local file.
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

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
        fmt.Printf("Saving credential file to: %s\n", path)
        f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
        if err != nil {
                log.Fatalf("Unable to cache oauth token: %v", err)
        }
        defer f.Close()
        json.NewEncoder(f).Encode(token)
}

func main() {
        data := "[]"
        file, err := os.Create("pipe.json")
        if err != nil {
                log.Fatalf("Error creating file: %v", err)
        }
        defer file.Close()
        _, _ = file.WriteString(data)

        
        ctx := context.Background()
        b, err := os.ReadFile("credentials.json")
        if err != nil {
                log.Fatalf("Unable to read client secret file: %v", err)
        }

        // If modifying these scopes, delete your previously saved token.json.
        config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
        if err != nil {
                log.Fatalf("Unable to parse client secret file to config: %v", err)
        }
        client := getClient(config)

        srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
        if err != nil {
                log.Fatalf("Unable to retrieve Gmail client: %v", err)
        }
        
        user := os.Args[1]

        // Perform the list request with a query
        r, err := srv.Users.Messages.List(user).Q("from:" + os.Args[2]).MaxResults(9).Do()
        if err != nil {
                log.Fatalf("Unable to retrieve messages: %v", err)
        }

        var jsonData []map[string]string
        // Iterate through the list of messages
        for _, msg := range r.Messages {
                // Get the full message details using the message ID
                fullMsg, err := srv.Users.Messages.Get(user, msg.Id).Do()
                if err != nil {
                        log.Fatalf("Unable to retrieve message details: %v", err)
                }

                sender := ""
                for _, header := range fullMsg.Payload.Headers {
                        if header.Name == "From" {
                                sender = header.Value
                                break
                        }
                }
                // Print or process the sender address

                subject := ""
                for _, header := range fullMsg.Payload.Headers {
                        if header.Name == "Subject" {
                                subject = header.Value
                                break
                        }
                }

                
                // Decode the message body
                body, err := decodeMessageBody(fullMsg.Payload)
                if err != nil {
                        log.Fatalf("Unable to decode message body: %v", err)
                }

                msgData := map[string]string{
                        "Id":      msg.Id,
                        "Sender":  sender,
                        "Subject": subject,
                        "Body":    body,
                }
                // Append the message data to the slice
                jsonData = append(jsonData, msgData)
        }
         // Encode the slice to JSON
        jsonBytes, err := json.Marshal(jsonData)
        if err != nil {
                log.Fatalf("Error encoding JSON: %v", err)
        }

        // Create a new file for writing
        file, err = os.Create("pipe.json")
        if err != nil {
                log.Fatalf("Error creating file: %v", err)
        }
        defer file.Close()

        // Write the JSON data to the file
        _, err = file.Write(jsonBytes)
        if err != nil {
                log.Fatalf("Error writing JSON to file: %v", err)
        }
}

// Function to decode message body
func decodeMessageBody(payload *gmail.MessagePart) (string, error) {
    if payload == nil {
        return "", fmt.Errorf("Payload is nil")
    }

    // Check if the body is available
    if payload.Body != nil && payload.Body.Data != "" {
        // Decode base64-encoded body
        bodyBytes, err := base64.URLEncoding.DecodeString(payload.Body.Data)
        if err != nil {
            return "", fmt.Errorf("Failed to decode message body: %v", err)
        }
        return string(bodyBytes), nil
    }

    // If body is not available, recursively check the parts
    for _, part := range payload.Parts {
        if part != nil {
            return decodeMessageBody(part)
        }
    }

    return "", fmt.Errorf("No message body found")
}