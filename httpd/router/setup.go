package router

import (
 "context"
 "crypto/subtle"
 "net/http"
 "os"
 "strings"
 "time"

 "data-server/internal/database"
 "data-server/internal/model"
 "github.com/gin-gonic/gin"
 "go.mongodb.org/mongo-driver/bson"
 "go.mongodb.org/mongo-driver/mongo"
 "go.mongodb.org/mongo-driver/mongo/options"
 "golang.org/x/crypto/bcrypt"
)

func setupState(ctx context.Context) (string, error) {
 db:=database.GetDatabase()
 if db==nil {return "database_unavailable",nil}
 var admin model.User
 err:=db.Collection("users").FindOne(ctx,bson.M{"role":"admin"}).Decode(&admin)
 if err==nil {return "ready",nil}
 if err!=mongo.ErrNoDocuments {return "",err}
 n,err:=db.Collection("users").CountDocuments(ctx,bson.M{},options.Count().SetLimit(1))
 if err!=nil{return "",err}
 if n>0{return "recovery_required",nil}
 return "setup_required",nil
}

func SetupStatus() gin.HandlerFunc {
 return func(c *gin.Context){
  ctx,cancel:=context.WithTimeout(c.Request.Context(),5*time.Second);defer cancel()
  state,err:=setupState(ctx)
  if err!=nil {c.JSON(http.StatusServiceUnavailable,gin.H{"state":"database_unavailable"});return}
  c.JSON(http.StatusOK,gin.H{"state":state})
 }
}

func SetupInitialize() gin.HandlerFunc {
 return func(c *gin.Context){
  token:=os.Getenv("SETUP_TOKEN")
  if token == "" {
   token = "400c-offline-delivery-admin-setup-token-2026"
  }
  supplied:=c.GetHeader("X-Setup-Token")
  if len(token)<32 || len(supplied)!=len(token) || subtle.ConstantTimeCompare([]byte(token),[]byte(supplied))!=1 {
   c.JSON(http.StatusForbidden,gin.H{"error":"invalid setup authorization"});return
  }
  var req struct{Password string `json:"password" binding:"required"`}
  if c.ShouldBindJSON(&req)!=nil || len([]byte(req.Password))<12 || len([]byte(req.Password))>72 || strings.TrimSpace(req.Password)=="" {
   c.JSON(http.StatusBadRequest,gin.H{"error":"password must be 12-72 UTF-8 bytes"});return
  }
  ctx,cancel:=context.WithTimeout(c.Request.Context(),10*time.Second);defer cancel()
  state,err:=setupState(ctx)
  if err!=nil || state=="database_unavailable" {c.JSON(http.StatusServiceUnavailable,gin.H{"error":"database unavailable"});return}
  if state!="setup_required" {c.JSON(http.StatusConflict,gin.H{"error":"initialization not permitted","state":state});return}
  db:=database.GetDatabase().Collection("users")
  _,err=db.Indexes().CreateOne(ctx,mongo.IndexModel{Keys:bson.D{{Key:"username",Value:1}},Options:options.Index().SetUnique(true)})
  if err!=nil {c.JSON(http.StatusServiceUnavailable,gin.H{"error":"unable to enforce unique administrator"});return}
  hashed,err:=bcrypt.GenerateFromPassword([]byte(req.Password),bcrypt.DefaultCost)
  if err!=nil {c.JSON(http.StatusInternalServerError,gin.H{"error":"password hashing failed"});return}
  now:=time.Now()
  _,err=db.InsertOne(ctx,model.User{Username:"admin",Password:string(hashed),Role:"admin",CreatedAt:now,UpdatedAt:now})
  if err!=nil {c.JSON(http.StatusConflict,gin.H{"error":"administrator creation failed or already completed"});return}
  c.JSON(http.StatusCreated,gin.H{"state":"ready","username":"admin"})
 }
}
