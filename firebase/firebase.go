package firebase

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"errors"
	storage2 "firebase.google.com/go/storage"
	"fmt"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/google/uuid"
	"golang.org/x/oauth2/google"
	"io"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/messaging"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

//FireAuth is the object holding the initialized firebase app
type FireAuth struct {
	app    *firebase.App
	client *auth.Client
}

type UserInfoFirebase struct {
	UserID string
	AuthID string
}

//This is the key for authID in the jwt
const userExternalIDKey = "user_id"

var FireAuthInstance *FireAuth
var FirebaseClient *messaging.Client
var FirebaseStorageClient *storage2.Client
var fireKey string

//NewFirebase creates a new instance of a firebase app to use
func init() {
	fireKey = os.Getenv("FIREBASE_KEY")

	opt := option.WithCredentialsJSON([]byte(fireKey))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
	fireAuth, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("error initializing fireAuth: %v\n", err)
	}

	ctx := context.Background()
	client, err := app.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}

	ctx = context.Background()
	storageClient, err := app.Storage(ctx)
	if err != nil {
		log.Fatalf("error getting Storage client: %v\n", err)
	}

	FireAuthInstance = &FireAuth{app: app, client: fireAuth}
	FirebaseClient = client
	FirebaseStorageClient = storageClient
}

func (fAuth *FireAuth) GetFirebaseUserID(ctx context.Context, phoneNumber, email string) (string, error) {
	if phoneNumber != "" {
		user, err := fAuth.client.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Error(fmt.Sprintf("error getting User: %v", err))
			return "", err
		}
		return user.UID, nil
	} else if email != "" {
		user, err := fAuth.client.GetUserByEmail(ctx, email)
		if err != nil {
			log.Error(fmt.Sprintf("error getting User: %v", err))
			return "", err
		}
		return user.UID, nil
	}
	return "", errors.New("phone and email both are null")
}

func (fAuth *FireAuth) CreateUser(phoneNumber, email string) (*auth.UserRecord, error) {
	params := new(auth.UserToCreate)

	if phoneNumber != "" {
		params = params.PhoneNumber(phoneNumber)
	}
	if email != "" {
		params = params.Email(email)
	}

	u, err := fAuth.client.CreateUser(context.Background(), params)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (fAuth *FireAuth) VerifyToken(rawToken string) (*auth.Token, error) {
	if len(rawToken) == 0 {
		return nil, errors.New("JWT token not found in header")
	}

	token := strings.TrimPrefix(rawToken, "Bearer ")

	authToken, err := fAuth.client.VerifyIDToken(context.Background(), token)
	if err != nil {
		return nil, err
	}

	return authToken, nil
}

func GetAuthId(token *auth.Token) (string, error) {
	if token == nil {
		newErr := errors.New("failed to add firebase info to context")
		log.Errorf("error adding firebase info to context: %v", newErr)
		return "", newErr
	}

	var authID string
	for key, value := range token.Claims {
		switch key {
		case userExternalIDKey:
			authID = value.(string)
		}
	}

	return authID, nil
}

// CustomTokenAuth provides a firebase custom token for the driver to use to authenticate
func (fAuth *FireAuth) CustomTokenAuth(ctx context.Context, userExternalID string) (string, error) {
	customToken, err := fAuth.client.CustomToken(ctx, userExternalID)
	if err != nil {
		log.Error(fmt.Sprintf("error generating custom token: %v\n", err))
		return "", err
	}
	return customToken, nil
}

// UploadToFirebase uploads image to firebase
func UploadToFirebase(file []byte, downloadedFileName string) (string, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Minute*50)
	defer cancel()

	uuID := uuid.New().String()
	uploadedFileName := uuID + downloadedFileName

	bucket, err := FirebaseStorageClient.Bucket(models.BucketLink)
	if err != nil {
		return "", err
	}

	object := bucket.Object(uploadedFileName)
	writer := object.NewWriter(ctx)
	if _, err := io.Copy(writer, bytes.NewReader(file)); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return uploadedFileName, nil
}

// GetURL returns public URL of given bucket and path of an image
func GetURL(imageInfo *models.Image) (string, error) {
	cfg, err := google.JWTConfigFromJSON([]byte(fireKey))
	if err != nil {
		return "", err
	}

	method := "GET"
	expires := time.Now().Add(time.Minute * 60)

	url, err := storage.SignedURL(imageInfo.Bucket, imageInfo.Path, &storage.SignedURLOptions{
		GoogleAccessID: cfg.Email,
		PrivateKey:     cfg.PrivateKey,
		Method:         method,
		Expires:        expires,
	})
	if err != nil {
		return "", err
	}
	return url, nil
}
