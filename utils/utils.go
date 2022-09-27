package utils

import (
	"crypto/sha512"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var generator *shortid.Shortid

type clientError struct {
	ID            string `json:"id"`
	MessageToUser string `json:"messageToUser"`
	DeveloperInfo string `json:"developerInfo"`
	Err           string `json:"error"`
	StatusCode    int    `json:"statusCode"`
	IsClientError bool   `json:"isClientError"`
}

func init() {
	g, err := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	if err != nil {
		logrus.Panicf("Failed to initialize utils package with error: %+v", err)
	}
	generator = g
}

// ParseBody parses the values from io reader to a given interface
func ParseBody(body io.Reader, out interface{}) error {
	err := json.NewDecoder(body).Decode(out)
	if err != nil {
		return err
	}
	return nil
}

// EncodeJSONBody writes the JSON body to response writer
func EncodeJSONBody(resp http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(resp).Encode(data)
}

// RespondJSON sends the interface as a JSON
func RespondJSON(w http.ResponseWriter, statusCode int, body interface{}) {
	w.WriteHeader(statusCode)
	if body != nil {
		if err := EncodeJSONBody(w, body); err != nil {
			logrus.Errorf("Failed to respond JSON with error: %+v", err)
		}
	}
}

// newClientError creates structured client error response message
func newClientError(err error, statusCode int, messageToUser string, additionalInfoForDevs ...string) *clientError {
	additionalInfoJoined := strings.Join(additionalInfoForDevs, "\n")
	if len(additionalInfoJoined) == 0 {
		additionalInfoJoined = messageToUser
	}

	errorID, _ := generator.Generate()
	var errString string
	if err != nil {
		errString = err.Error()
	}
	return &clientError{
		ID:            errorID,
		MessageToUser: messageToUser,
		DeveloperInfo: additionalInfoJoined,
		Err:           errString,
		StatusCode:    statusCode,
		IsClientError: true,
	}
}

// RespondError sends an error message to the API caller and logs the error
func RespondError(w http.ResponseWriter, statusCode int, err error, messageToUser string, additionalInfoForDevs ...string) {
	logrus.Errorf("status: %d, message: %s, err: %+v ", statusCode, messageToUser, err)
	clientError := newClientError(err, statusCode, messageToUser, additionalInfoForDevs...)
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(clientError); err != nil {
		logrus.Errorf("Failed to send error to caller with error: %+v", err)
	}
}

// HashString generates SHA256 for a given string
func HashString(toHash string) string {
	sha := sha512.New()
	sha.Write([]byte(toHash))
	return hex.EncodeToString(sha.Sum(nil))
}

// StringToInt converts given string to int type
func StringToInt(str string) (int, error) {
	val, err := strconv.Atoi(str)
	if err != nil {
		return -1, err
	}
	return val, nil
}

//GenerateOTP generates and returns a random number less the 9999
func GenerateOTP() int {
	max, min := 9999, 1000
	rand.Seed(time.Now().UnixNano())
	randomNum := rand.Intn(max-min) + min
	return randomNum
}

func GeoDistance(lng1 float64, lat1 float64, lng2 float64, lat2 float64) float64 {
	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * lat1 / 180)
	radlat2 := float64(PI * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515
	dist = dist * 1.609344

	return dist
}

//GetOffsetLimit return offset and limit and error if any
func GetOffsetLimit(r *http.Request) (int, int, error) {
	var (
		offset = 0
		limit  = 10
		err    error
	)
	if r.URL.Query().Get("offset") != "" {
		offset, err = strconv.Atoi(r.URL.Query().Get("offset"))
	}

	if r.URL.Query().Get("limit") != "" {
		limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
	}
	return offset, limit, err
}

// ReadFromFile reads image file and returns required info
func ReadFromFile(r *http.Request, imageType string) (multipart.File, []byte, string, error) {
	file, handler, err := r.FormFile(imageType)
	if err != nil {
		return nil, nil, "", err
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return nil, nil, "", err
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, nil, "", err
	}
	return file, fileBytes, handler.Filename, nil
}

func GenerateJWT(id int, email string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["userId"] = id
	claims["email"] = email
	claims["exp"] = time.Now().Add(time.Hour * 24 * 30).Unix()
	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET_KEY")))
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return tokenString, nil
}

func CreateCsvOfOrders(orders []models.ScheduledOrderCsv) (string, error) {
	data := make([][]string, 0)
	csvHeader := []string{"orderId", "mode", "userName", "userPhone", "address", "deliveryTime", "staffName", "staffPhone", "createdAt", "startDate", "endDate"}
	for i := 1; i < 51; i++ {
		csvHeader = append(csvHeader, "itemName-(quantity * baseQuantity)")
	}
	data = append(data, csvHeader)
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return "", err
	}
	for _, order := range orders {
		record := []string{
			strconv.Itoa(order.OrderID),
			order.OrderMode,
			order.UserName, order.UserPhone, order.UserAddress, order.DeliveryTime.In(loc).Format(time.RFC850),
			order.StaffName.String, order.StaffPhone.String,
			order.CreatedAt.In(loc).Format(time.RFC850), order.StartDate.In(loc).Format(time.RFC850), order.EndDate.In(loc).Format(time.RFC850)}
		for _, item := range order.Items {
			record = append(record, fmt.Sprintf("%s - (%d * %s)", item.Name, item.Quantity, item.BaseQuantity))
		}
		data = append(data, record)
	}

	fileName := fmt.Sprintf("file_%d.csv", time.Now().UnixNano())
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.WriteAll(data)
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func CreateCsvOfOrderHistory(orders []models.ScheduledOrderCsv) (string, error) {
	data := make([][]string, 0)
	csvHeader := []string{"orderId", "mode", "userName", "userPhone", "address", "deliveryTime", "status", "staffName", "staffPhone", "createdAt"}
	for i := 1; i < 51; i++ {
		csvHeader = append(csvHeader, "itemName-(quantity * baseQuantity)")
	}
	data = append(data, csvHeader)
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return "", err
	}
	for _, order := range orders {
		record := []string{
			strconv.Itoa(order.OrderID),
			order.OrderMode,
			order.UserName, order.UserPhone, order.UserAddress, order.DeliveryTime.In(loc).Format(time.RFC850),
			order.Status,
			order.StaffName.String, order.StaffPhone.String,
			order.CreatedAt.In(loc).Format(time.RFC850)}
		for _, item := range order.Items {
			record = append(record, fmt.Sprintf("%s - (%d * %s)", item.Name, item.Quantity, item.BaseQuantity))
		}
		data = append(data, record)
	}

	fileName := fmt.Sprintf("file_%d.csv", time.Now().UnixNano())
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.WriteAll(data)
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func CreateCsvOfUserStats(userStats []models.DashBoardUserDetails) (string, error) {
	data := make([][]string, 0)
	csvHeader := []string{"userName", "userPhone", "primaryLocation", "registrationDate", "totalOrders", "Denied", "Canceled", "avgRating", "flagged"}
	data = append(data, csvHeader)
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return "", err
	}
	for _, user := range userStats {
		record := []string{
			user.Name, user.Contact, user.DefaultAddress.String, user.RegDate.In(loc).Format(time.RFC850),
			strconv.Itoa(user.TotalOrders), strconv.Itoa(user.DeniedOrders), strconv.Itoa(user.CanceledOrders),
			strconv.Itoa(int(user.AvgRating)), strconv.Itoa(user.FlagCount)}
		data = append(data, record)
	}

	fileName := fmt.Sprintf("user_stats_data_%d.csv", time.Now().UnixNano())
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.WriteAll(data)
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func FilterOrderItems(Items []models.ItemInfo) []models.ItemInfo {
	if len(Items) == 0 {
		return []models.ItemInfo{}
	}
	newItems := make([]models.ItemInfo, 0)
	for i := range Items {
		item := Items[i]
		exist := false
		for j := range newItems {
			if newItems[j].Id == item.Id {
				exist = true
			}
		}
		if !exist {
			newItems = append(newItems, item)
		}
	}
	return newItems

}
