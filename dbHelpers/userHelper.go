package dbHelpers

import (
	"database/sql"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"time"
)

// IsUserExist checks if an user exists with a given phone
func IsUserExist(authID string) (int, error) {
	SQL := `SELECT id FROM users WHERE auth_id=$1 AND archived_at IS NULL`
	var id int
	err := database.YourDailyDB.Get(&id, SQL, authID)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, nil
}

// InsertUser creates a new user entry
func InsertUser(name, phone, authId string, permission models.UserPermission) (int, error) {
	var userID int
	txError := database.Tx(func(tx *sqlx.Tx) error {
		SQL := `INSERT INTO users(name, phone, auth_id) VALUES ($1, $2, $3) RETURNING id`
		err := tx.Get(&userID, SQL, name, phone, authId)
		if err != nil {
			return err
		}

		SQL = `INSERT INTO user_permission(user_id, permission_type) VALUES ($1, $2)`
		_, err = tx.Exec(SQL, userID, permission)
		if err != nil {
			return err
		}

		SQL = `INSERT INTO fcm_token(user_id, created_at) VALUES($1, $2)`
		_, err = tx.Exec(SQL, userID, time.Now())
		if err != nil {
			return err
		}

		return nil
	})
	return userID, txError
}

// UserPermissionById returns arrays of permissions for given user
func UserPermissionById(userID int) ([]models.UserPermission, error) {
	SQL := `SELECT permission_type FROM user_permission WHERE user_id=$1`
	permissions := make([]models.UserPermission, 0)
	err := database.YourDailyDB.Select(&permissions, SQL, userID)
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// ModifyUser modifies a given user entry
func ModifyUser(name, email string, userId int) error {
	SQL := `UPDATE users SET name = $1, email = $2, updated_at = $3 WHERE id = $4`
	_, err := database.YourDailyDB.Exec(SQL, name, email, time.Now(), userId)
	return err
}

// GetUserById returns the leave details for a given id
func GetUserById(userId int) (*models.User, error) {
	SQL := `SELECT 
				id,
				name,
       			phone,
       			email,
       			profile_image,
				created_at
			FROM users
			WHERE  archived_at IS NULL
			AND id = $1`

	var user models.User

	err := database.YourDailyDB.Get(&user, SQL, userId)
	if err != nil {
		return nil, err
	}
	if user.ProfileImageID.Valid {
		user.ProfileImageLink, err = GetImageUrl(user.ProfileImageID.Int)
		if err != nil {
			logrus.Errorf("GetUserById: failed to fetch image link err :%v", err)
		}
	}
	return &user, nil

}

// GetUserByAuthId returns the user details for a given authId
func GetUserByAuthId(authId string) (*models.User, error) {
	SQL := `SELECT 
				id,
				name, 
				phone, 
				email, 
				profile_image,
				created_at 
			FROM users
			WHERE archived_at IS NULL
			AND auth_id = $1`
	var user models.User
	if err := database.YourDailyDB.Get(&user, SQL, authId); err != nil {
		return nil, err
	}

	return &user, nil
}

func IsPhoneExist(phone string) (int, error) {
	SQL := `SELECT 
				id
			FROM users
			WHERE archived_at IS NULL
			AND phone = $1`

	var userID int
	err := database.YourDailyDB.Get(&userID, SQL, phone)

	if err != nil {
		return -1, err
	}
	return userID, nil
}

// ModifyFcmToken updates fcm token for given user id
func ModifyFcmToken(token string, userID int) error {
	SQL := `UPDATE fcm_token SET token = $1, updated_at = $2 WHERE user_id = $3`
	_, err := database.YourDailyDB.Exec(SQL, token, time.Now(), userID)
	return err
}

//GetStaffCount return the count of user according to roles
func GetStaffCount(staffType string) ([]int, error) {
	query := `SELECT id
				FROM users
        		 JOIN user_permission up ON users.id = up.user_id
						WHERE up.permission_type =$1 AND archived_at IS NULL`
	var staffIDs = make([]int, 0)
	err := database.YourDailyDB.Select(&staffIDs, query, staffType)
	return staffIDs, err
}
func GerUserCount() ([]int, error) {
	query := `
			SELECT id
				FROM users
				WHERE id NOT IN (
					SELECT u.id
					FROM users u
							 JOIN user_permission up ON u.id = up.user_id
					WHERE up.permission_type <> 'user'
				) AND archived_at IS NULL
`
	userIds := make([]int, 0)
	err := database.YourDailyDB.Select(&userIds, query)
	return userIds, err
}

//GetActiveUsers return the count of users who have placed order in last 10 days
func GetActiveUsers() (int, error) {
	var OnGoing int
	query := `SELECT COUNT(DISTINCT (user_id))
FROM users
         JOIN orders o ON users.id = o.user_id
WHERE o.created_at > NOW() - '10 DAYS'::INTERVAL;`
	err := database.YourDailyDB.Get(&OnGoing, query)
	return OnGoing, err
}

// StoreProfileImage stores profile image info of a user
func StoreProfileImage(bucket string, path string, imageType models.ImageType, userID int) error {
	txError := database.Tx(func(tx *sqlx.Tx) error {

		// if user already has an profile image, archive it
		SQL := `UPDATE images
			SET archived_at = $1
			WHERE archived_at IS NULL
			AND id = (SELECT profile_image FROM users WHERE id = $2)`

		_, err := tx.Exec(SQL, time.Now(), userID)
		if err != nil {
			return err
		}

		// store image info
		SQL = `INSERT INTO images(type, bucket, path, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
		var imageID int
		err = tx.Get(&imageID, SQL, string(imageType), bucket, path, time.Now())
		if err != nil {
			return err
		}

		// update foreign key
		SQL = `UPDATE users SET profile_image = $1 WHERE id=$2 AND archived_at IS NULL`
		_, err = tx.Exec(SQL, imageID, userID)
		if err != nil {
			return err
		}
		return nil
	})
	return txError
}

// GetImageInfoByUserID returns image info - bucket & path for given userid
func GetImageInfoByUserID(userID int) (*models.Image, error) {
	SQL := `SELECT u.bucket, u.path
		FROM images u
		JOIN users us
		ON u.id = us.profile_image
		WHERE us.id = $1`

	var imageInfo models.Image
	err := database.YourDailyDB.Get(&imageInfo, SQL, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &imageInfo, nil
}

func CheckIfEmailIsRegisteredToSomeoneElse(email string, userID int) (int, error) {
	SQL := `
		SELECT COUNT(*)
		FROM users
		WHERE email = $1
		  AND id <> $2
`
	var exits int
	err := database.YourDailyDB.Get(&exits, SQL, email, userID)
	return exits, err
}

func GetFlagCountAndLastOrderStatus(userId int) (int, models.OrderStatus, error) {
	//language=sql
	query := `
		SELECT flags, status
		FROM orders o
				 JOIN users u ON o.user_id = u.id
		WHERE user_id = $1
		  AND o.created_at >= u.unflagged_at
		  AND status IN ('delivered', 'cancelled')
		ORDER BY o.created_at DESC
		LIMIT 1
`
	var flagCount int
	var status models.OrderStatus
	err := database.YourDailyDB.QueryRowx(query, userId).Scan(&flagCount, &status)
	if err != nil && err != sql.ErrNoRows {
		return flagCount, status, err
	}
	return flagCount, status, nil
}

func SetFlag(flagCount, userID int) error {
	//language=sql
	query := `
		UPDATE users
		SET flags = $1
		WHERE id = $2
`
	_, err := database.YourDailyDB.Exec(query, flagCount, userID)
	return err
}

func SetUnFlag(userID int) error {
	//language=sql
	query := `
		UPDATE users
		SET flags=0,
			unflagged_at = NOW()
		WHERE id = $1;
`
	_, err := database.YourDailyDB.Exec(query, userID)
	return err
}
