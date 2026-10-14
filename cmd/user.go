package cmd

import (
	"context"
	"fmt"
	"data-server/internal/database"
	"data-server/internal/model"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	userCmd = &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	userInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Create initial administrator only when none exists",
		RunE:  runUserInit,
	}

	userStatusCmd = &cobra.Command{ Use: "status", Short: "Check whether an administrator exists", RunE: runUserStatus },

	userAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		Args:  cobra.ExactArgs(3),
		RunE:  runUserAdd,
	}
)

func initUserCommands() {
	// Add user init as subcommand
	userCmd.AddCommand(userInitCmd)
	userCmd.AddCommand(userStatusCmd)
	// Add user command to root
	rootCmd.AddCommand(userCmd)
	// Add user add command with required flags
	userCmd.AddCommand(userAddCmd)
}

func runUserStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	configureDatabaseFromEnv()
	database.InitializedDatabase(ctx)
	defer database.Stop()
	if database.GetDatabase() == nil {
		return fmt.Errorf("database unavailable")
	}
	var admin model.User
	err := database.FindOne("users", bson.M{"role": "admin"}).Decode(&admin)
	if err == nil {
		fmt.Println("ADMIN_EXISTS")
		return nil
	}
	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("administrator lookup failed: %w", err)
	}
	count, err := database.GetDatabase().Collection("users").CountDocuments(ctx, bson.M{}, options.Count().SetLimit(1))
	if err != nil { return fmt.Errorf("count users: %w", err) }
	if count > 0 {
		return fmt.Errorf("existing users without administrator; manual recovery required")
	}
	fmt.Println("INITIAL_SETUP_REQUIRED")
	return nil
}

func runUserInit(cmd *cobra.Command, args []string) error {
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 configureDatabaseFromEnv()
 database.InitializedDatabase(ctx)
 defer database.Stop()
 if err := initializeDefaultUser(ctx); err != nil {
  return fmt.Errorf("initialize administrator: %w", err)
 }
 logrus.Info("Administrator initialization check completed")
 return nil
}

func runUserAdd(cmd *cobra.Command, args []string) error {
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 configureDatabaseFromEnv()
 database.InitializedDatabase(ctx)
 defer database.Stop()
 if database.GetDatabase() == nil { return fmt.Errorf("database unavailable") }
 if len(args[1]) < 12 || len([]byte(args[1])) > 72 { return fmt.Errorf("password must be at least 12 bytes and at most 72 UTF-8 bytes") }
 if args[2] != "admin" && args[2] != "user" { return fmt.Errorf("role must be admin or user") }
 if err := createUser(ctx, args[0], args[1], "", args[2]); err != nil { return fmt.Errorf("add user: %w", err) }
 logrus.Info("User created")
 return nil
}

func configureDatabaseFromEnv() {
	if databaseURI := os.Getenv("DATABASE_URI"); databaseURI != "" {
		viper.Set("database.uri", databaseURI)
	}
}

func initializeDefaultUser(ctx context.Context) error {
	if database.GetDatabase() == nil {
		return fmt.Errorf("database unavailable; refusing administrator initialization")
	}
	var existing model.User
	err := database.FindOne("users", bson.M{"role": "admin"}).Decode(&existing)
	if err == nil {
		logrus.Info("Administrator already exists; preserving accounts and passwords")
		return nil
	}
	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("check administrator: %w", err)
	}
	// Refuse to bootstrap over an existing nonempty user database.
	count, err := database.GetDatabase().Collection("users").CountDocuments(ctx, bson.M{}, options.Count().SetLimit(1))
	if err != nil { return fmt.Errorf("check existing users: %w", err) }
	if count > 0 {
		return fmt.Errorf("users exist but no administrator; manual recovery required")
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if len([]byte(password)) < 12 || len([]byte(password)) > 72 {
		return fmt.Errorf("ADMIN_PASSWORD must be 12-72 UTF-8 bytes")
	}
	if err := createUser(ctx, "admin", password, "admin@localhost", "admin"); err != nil {
		return fmt.Errorf("create initial administrator: %w", err)
	}
	logrus.Info("Initial administrator created")
	return nil
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
