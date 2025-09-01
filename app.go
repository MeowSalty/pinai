package main

import (
	"flag"

	"github.com/gofiber/fiber/v2"
)

var (
	port = flag.String("port", ":3000", "监听端口")
	prod = flag.Bool("prod", false, "在生产环境中启用 prefork")
)

func main() {
	app := fiber.New(fiber.Config{
		Prefork: *prod, // go run app.go -prod
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Listen(*port)
}
