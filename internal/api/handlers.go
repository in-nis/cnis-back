package api

import (
    "context"
	"log"
    "net/http"
	"strconv"
	"strings"
	"sort"

    "github.com/gin-gonic/gin"
    "github.com/in-nis/cnis-back/internal/db"
	"github.com/in-nis/cnis-back/internal/models"
	"github.com/in-nis/cnis-back/internal/excel"
)

// UpdateUserGradeRequest is the request body for updating grade
type UpdateUserGradeRequest struct {
    Grade       int    `json:"grade"`
    GradeLetter string `json:"grade_letter"`
}

// UpdateUserGrade godoc
// @Summary      Update user's grade
// @Description  Updates the grade and grade letter for the authenticated user
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        body  body  UpdateUserGradeRequest  true  "Grade info"
// @Success      200   {object} map[string]string
// @Failure      400   {object} map[string]string
// @Failure      500   {object} map[string]string
// @Security     BearerAuth
// @Router       /user/grade [patch]
func UpdateUserGrade(c *gin.Context) {
    email := c.GetString("email")

    var req UpdateUserGradeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    if err := db.UpdateUserGrade(context.Background(), email, req.Grade, req.GradeLetter); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update grade"})
        return
    }

    c.JSON(200, gin.H{"message": "Grade updated"})
}

// GetUserGroups godoc
// @Summary      Get user's groups
// @Description  Returns all groups for the authenticated user
// @Tags         user
// @Produce      json
// @Success      200   {array}  map[string]interface{}
// @Failure      500   {object} map[string]string
// @Security     BearerAuth
// @Router       /user/groups [get]
func GetUserGroups(c *gin.Context) {
    email := c.GetString("email")
    groups, err := db.GetUserGroups(context.Background(), email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
        return
    }
    c.JSON(200, groups)
}

// AddUserGroupRequest is the request body for adding a group
type AddUserGroupRequest struct {
    LessonName  string `json:"lesson_name"`
    LessonGroup string `json:"lesson_group"`
}

// AddUserGroup godoc
// @Summary      Add a user group
// @Description  Adds a new group for the authenticated user
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        body  body  AddUserGroupRequest  true  "Group info"
// @Success      200   {object} map[string]string
// @Failure      400   {object} map[string]string
// @Failure      500   {object} map[string]string
// @Security     BearerAuth
// @Router       /user/groups [post]
func AddUserGroup(c *gin.Context) {
    email := c.GetString("email")

    var req AddUserGroupRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    if err := db.AddUserGroup(context.Background(), email, req.LessonName, req.LessonGroup); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add group"})
        return
    }

    c.JSON(200, gin.H{"message": "Group added"})
}

// DeleteUserGroup godoc
// @Summary      Delete a user group
// @Description  Deletes a group by ID for the authenticated user
// @Tags         user
// @Produce      json
// @Param        id   path  int  true  "Group ID"
// @Success      200  {object} map[string]string
// @Failure      500  {object} map[string]string
// @Security     BearerAuth
// @Router       /user/groups/{id} [delete]
func DeleteUserGroup(c *gin.Context) {
    email := c.GetString("email")
    id := c.Param("id")

    if err := db.DeleteUserGroup(context.Background(), email, id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
        return
    }

    c.JSON(200, gin.H{"message": "Group deleted"})
}

// UserProfileResponse is a safe version of User for API responses
type UserProfileResponse struct {
    ID          uint                `json:"id"`
    Email       string              `json:"email"`
    Grade       int                 `json:"grade"`
    GradeLetter string              `json:"grade_letter"`
    Groups      []models.UserGroup  `json:"groups"`
}

// GetMe godoc
// @Summary      Get current user profile
// @Description  Returns the authenticated user's profile
// @Tags         user
// @Produce      json
// @Success      200 {object} UserProfileResponse
// @Failure      401 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /me [get]
func GetMe(c *gin.Context) {
    email := c.GetString("email")

    user, err := db.GetUserByEmail(context.Background(), email)
    if err != nil {
        log.Println(err)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }

    // Map DB user → safe response
    resp := UserProfileResponse{
        ID:          user.ID,
        Email:       user.Email,
        Grade:       user.Grade,
        GradeLetter: user.GradeLetter,
        Groups:      user.Groups,
    }

    c.JSON(http.StatusOK, resp)
}

// ParseLessons godoc
// @Summary      Parse Excel and save lessons
// @Description  Parses sheet.xlsx and saves lessons into DB
// @Tags         lessons
// @Produce      json
// @Success      200 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /lessons/parse [post]
func ParseLessons(c *gin.Context) {
	path := "sheet.xlsx"

	if err := db.DeleteAllLessons(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear lessons"})
		return
	}

	lessons, err := excel.ParseExcel(path)
	if err != nil {
		log.Println("❌ Failed to parse Excel:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Excel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lessons parsed and saved", "count": len(lessons)})
}

// GetLessonsByClassAndGroups godoc
// @Summary      Get lessons by class and multiple groups
// @Description  Returns lessons filtered by grade+letter and a list of lessonName+lessonGroup pairs
// @Tags         lessons
// @Accept       json
// @Produce      json
// @Param        grade   query  int    true  "Grade (parallel)"
// @Param        letter  query  string true  "Grade letter"
// @Param        body    body   []models.LessonGroupFilter  true  "List of lessonName+lessonGroup filters"
// @Success      200 {array} models.Lesson
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /lessons/filter [post]
func GetLessonsByClassAndGroups(c *gin.Context) {
	gradeStr := c.Query("grade")
	letter := c.Query("letter")

	if gradeStr == "" || letter == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing grade or letter"})
		return
	}

	grade, err := strconv.Atoi(gradeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade"})
		return
	}

	// Parse filters from query ?q=LessonName:LessonGroup
	qParams := c.QueryArray("q")
	var filters []models.LessonGroupFilter
	for _, q := range qParams {
		parts := strings.SplitN(q, ":", 2)
		if len(parts) == 2 {
			filters = append(filters, models.LessonGroupFilter{
				LessonName:  parts[0],
				LessonGroup: parts[1],
			})
		} else if len(parts) == 1 {
			filters = append(filters, models.LessonGroupFilter{
				LessonName:  parts[0],
				LessonGroup: "",
			})
		}
	}

	log.Println(filters)

	lessons, err := db.GetLessonsByClassAndGroups(context.Background(), grade, letter, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch lessons"})
		return
	}

	// Group lessons by LessonDay
	grouped := make(map[int][]models.Lesson)
	for _, l := range lessons {
		grouped[l.LessonDay] = append(grouped[l.LessonDay], l)
	}

	// Sort each day's lessons by LessonStart
	for day := range grouped {
		sort.Slice(grouped[day], func(i, j int) bool {
			return grouped[day][i].LessonStart.Before(grouped[day][j].LessonStart)
		})
	}

	c.JSON(http.StatusOK, grouped)
}