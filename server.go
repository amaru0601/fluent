package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	. "fluent/chat"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin: func(r *http.Request) bool {
			log.Printf("%s %s%s %v\n", r.Method, r.Host, r.RequestURI, r.Proto)
			return r.Method == http.MethodGet
		},
	}
)

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Route => handler
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!\n")
	})

	chatManager := &ChatManager{
		chats: make(map[string]*Chat),
	}

	e.GET("/chats/:id", chatManager.handle)
	e.POST("/login", login)

	r := e.Group("/auth")
	{
		config := middleware.JWTConfig{
			Claims:     &jwtCustomClaims{},
			SigningKey: []byte("iosonic"),
		}
		r.Use(middleware.JWTWithConfig(config))
		r.GET("/", func(c echo.Context) error {
			return c.String(http.StatusOK, "Hello, World!\n")
		})
	}

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

type jwtCustomClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username != "jon" || password != "123456" {
		return echo.ErrUnauthorized
	}

	claims := &jwtCustomClaims{
		"username",
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 72).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//TODO: put signing key to env variable
	t, err := token.SignedString([]byte("iosonic"))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"token": t,
	})
}

type ChatManager struct {
	chats map[string]*Chat
}

// originalmente el chat id puede ser null o int
// si el chat id es null es porque el cliente esta empezando una nueva conversacion
// se debe crear un historial del chat en la base de datos
// si el chat id es int conseguir el historial y conseguir el chat desde el manager

func (manager *ChatManager) handle(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	chatID := c.Param("id")
	if err != nil {
		log.Fatalln("Error on websocket connection:", err.Error())
	}
	defer ws.Close()

	// conseguir el chat
	if room, ok := manager.chats[chatID]; ok {
		//conectar cliente con web socket
		//TODO: conseguir user desde JWT
		fmt.Println("chat saved ...")
		user := &User{
			Username: "jaoks",
			Conn:     ws,
			Global:   room,
		}

		room.Join <- user
		user.Read()

		//go chat.Run()
	} else {
		chat := &Chat{
			Users:    make(map[string]*User),
			Messages: make(chan *Message),
			Join:     make(chan *User),
			Leave:    make(chan *User),
		}

		//TODO: conseguir user desde JWT
		user := &User{
			Username: "amaru",
			Conn:     ws,
			Global:   chat,
		}
		manager.chats[chatID] = chat
		go chat.Run()

		fmt.Println("joining...")
		chat.Join <- user
		fmt.Println("joined user 1 ...")
		user.Read()
		fmt.Println("done ...")

	}

	return nil
}