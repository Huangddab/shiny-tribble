package cmd

import (
	"context"
	"data-server/internal/database"
	"data-server/internal/model"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var (
	userCmd = &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	userInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize default admin user",
		Run:   runUserInit,
	}

	userAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		Run:   runUserAdd,
	}
)

func initUserCommands() {
	// Add user init as subcommand
	userCmd.AddCommand(userInitCmd)
	// Add user command to root
	rootCmd.AddCommand(userCmd)
	// Add user add command with required flags
	userCmd.AddCommand(userAddCmd)
}

func runUserInit(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize database
	database.InitializedDatabase(ctx)
	defer database.Stop()

	logrus.Info("Starting initialization...")

	// Create or reset default admin user
	if err := initializeDefaultUser(ctx); err != nil {
		logrus.Fatalf("failed to initialize default user: %v", err)
	}

	logrus.Info("Initialization completed successfully")
}

func runUserAdd(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize database
	database.InitializedDatabase(ctx)
	defer database.Stop()

	// Create or reset normal user
	if err := createUser(ctx, args[0], args[1], "", args[2]); err != nil {
		logrus.Fatalf("failed to add new user: %v", err)
	}

	logrus.Info("Add user successfully")
}

func initializeDefaultUser(ctx context.Context) error {
	username := "admin"
	password := "admin"
	email := "admin@localhost"

	logrus.Infof("Checking for existing admin user...")

	// Try to find existing admin user
	result := database.FindOne("users", bson.M{"username": username})

	var existingUser model.User
	err := result.Decode(&existingUser)

	if err == mongo.ErrNoDocuments {
		// User doesn't exist, create new one
		logrus.Infof("Admin user not found, creating new one...")
		return createUser(ctx, username, password, email, "admin")
	} else if err != nil {
		return err
	}

	// User exists, update password
	logrus.Infof("Admin user found, updating password...")
	return updateUser(ctx, existingUser.ID, password, "admin")
}

func createUser(_ context.Context, username, password, email string, role string) error {
	// Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	user := model.User{
		Username:  username,
		Password:  string(hashedPassword),
		Email:     email,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result, err := database.InsertOne("users", user)
	if err != nil {
		return err
	}

	logrus.Infof("Admin user created successfully: %v", result.InsertedID)
	return nil
}

func updateUser(_ context.Context, userID any, newPassword string, role string) error {
	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	updateData := bson.M{
		"role":       role,
		"password":   string(hashedPassword),
		"updated_at": time.Now(),
	}

	result, err := database.UpdateByID("users", userID, bson.M{"$set": updateData})
	if err != nil {
		return err
	}

	logrus.Infof("Admin user password updated successfully: %d document(s) modified", result.ModifiedCount)
	return nil
}
