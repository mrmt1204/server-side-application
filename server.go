package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/mrmt1204/server-side-application/bot"
	"log"
	"net/http"

	"github.com/mrmt1204/server-side-application/controller"
	"github.com/mrmt1204/server-side-application/db"
	"github.com/mrmt1204/server-side-application/model"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// Server はAPIサーバーが実装された構造体です
type Server struct {
	db              *sql.DB
	Engine          *gin.Engine
	simpleBotStream chan *model.Message
	gachaBotStream  chan *model.Message
}

// NewServer は新しいServerの構造体のポインタを返します
func NewServer() *Server {
	return &Server{
		Engine:          gin.Default(),
		simpleBotStream: make(chan *model.Message, 100),
		gachaBotStream:  make(chan *model.Message, 100),
	}
}

// Init はサーバーを初期化します
func (s *Server) Init(dbconf, env string) error {
	cs, err := db.NewConfigsFromFile(dbconf)
	if err != nil {
		return err
	}

	db, err := cs.Open(env)
	if err != nil {
		return err
	}
	s.db = db

	// routing
	s.Engine.LoadHTMLGlob("./templates/*")

	s.Engine.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	s.Engine.Static("/assets", "./assets")

	// api
	api := s.Engine.Group("/api")
	api.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	mctr := &controller.Message{DB: db, SimpleBotStream: s.simpleBotStream, GachaBotStream: s.gachaBotStream}
	api.GET("/messages", mctr.All)
	api.GET("/messages/:id", mctr.GetByID)
	api.POST("/messages", mctr.Create)
	api.PUT("/messages/:id", mctr.UpdateByID)

	return nil
}

// Close はDBとの接続を閉じてサーバーを終了します
func (s *Server) Close() error {
	return s.db.Close()
}

// Run はサーバーを起動します
func (s *Server) Run(port string) {
	// simple bot
	simpleBot := bot.SimpleBot{}
	go simpleBot.Run(s.simpleBotStream, fmt.Sprintf("http://0.0.0.0:%s", port))

	gachaBot := bot.GachaBot{}
	go gachaBot.Run(s.gachaBotStream, fmt.Sprintf("http://0.0.0.0:%s", port))
	err := s.Engine.Run(fmt.Sprintf(":%s", port))
	if err != nil {
		return
	}
}

func main() {
	var (
		dbconf = flag.String("dbconf", "dbconfig.yml", "database configuration file.")
		env    = flag.String("env", "development", "application envirionment (production, development etc.)")
		port   = flag.String("port", "8080", "listening port.")
	)
	flag.Parse()

	s := NewServer()
	if err := s.Init(*dbconf, *env); err != nil {
		log.Fatalf("fail to init server: %s", err)
	}
	defer s.Close()

	s.Run(*port)
}
