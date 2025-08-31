package db

import (
    "fmt"
    "log"
	"context"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "github.com/in-nis/cnis-back/internal/models"
)

var DB *gorm.DB

func InitDB(dsn string) {
    var err error
    DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatalf("failed to connect database: %v", err)
    }

    // AutoMigrate will create/update tables automatically
    err = DB.AutoMigrate(&models.Lesson{}, &models.User{}, &models.UserGroup{})
    if err != nil {
        log.Fatalf("failed to migrate database: %v", err)
    }

    fmt.Println("✅ Database connected and migrated")
}

func SaveLesson(ctx context.Context, l models.Lesson) error {
    return DB.WithContext(ctx).Create(&l).Error
}

func SaveLessons(ctx context.Context, lessons []models.Lesson) error {
    return DB.WithContext(ctx).Create(&lessons).Error
}

func DeleteAllLessons(ctx context.Context) error {
	return DB.WithContext(ctx).Where("1 = 1").Delete(&models.Lesson{}).Error
}

func SaveOrUpdateUser(ctx context.Context, u models.User) error {
    var existing models.User
    if err := DB.WithContext(ctx).Where("email = ?", u.Email).First(&existing).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return DB.WithContext(ctx).Create(&u).Error
        }
        return err
    }

    return DB.WithContext(ctx).Model(&existing).Updates(u).Error
}

func GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    var user models.User
    if err := DB.WithContext(ctx).Preload("Groups").Where("email = ?", email).First(&user).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

func PingDB() error {
	sqlDB, err := DB.DB() 
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func UpdateUserGrade(ctx context.Context, email string, grade int, letter string) error {
    return DB.WithContext(ctx).Model(&models.User{}).
        Where("email = ?", email).
        Updates(map[string]interface{}{"grade": grade, "grade_letter": letter}).Error
}

func GetUserGroups(ctx context.Context, email string) ([]models.UserGroup, error) {
    var user models.User
    if err := DB.WithContext(ctx).Preload("Groups").Where("email = ?", email).First(&user).Error; err != nil {
        return nil, err
    }
    return user.Groups, nil
}

func AddUserGroup(ctx context.Context, email, lessonName, lessonGroup string) error {
    var user models.User
    if err := DB.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
        return err
    }
    group := models.UserGroup{UserID: user.ID, LessonName: lessonName, LessonGroup: lessonGroup}
    return DB.WithContext(ctx).Create(&group).Error
}

func DeleteUserGroup(ctx context.Context, email, groupID string) error {
    var user models.User
    if err := DB.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
        return err
    }
    return DB.WithContext(ctx).Where("id = ? AND user_id = ?", groupID, user.ID).Delete(&models.UserGroup{}).Error
}

func GetLessonsByClassAndGroups(ctx context.Context, grade int, letter string, filters []models.LessonGroupFilter) ([]models.Lesson, error) {
	var lessons []models.Lesson

	tx := DB.WithContext(ctx).Where("grade = ? AND grade_letter = ?", grade, letter)
	for _, f := range filters {
		tx = tx.Or("grade = ? AND lesson_name = ? AND lesson_group = ?", grade, f.LessonName, f.LessonGroup)
	}

	if err := tx.Find(&lessons).Error; err != nil {
		return nil, err
	}
	return lessons, nil
}