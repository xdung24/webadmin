package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	_ "webadmin/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/swagger"
)

//go:embed dist/ng-matero/browser/*
var embeddedFiles embed.FS

// parse command-line flags
func parseFlags() (string, string, bool, bool, bool) {
	portFlag := flag.String("port", "", "Port to run the server on")
	sqliteFlag := flag.String("sqlite", "webadmin.db", "SQLite database file path")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose mode")
	metricsFlag := flag.Bool("metrics", false, "Enable metrics endpoint")
	swaggerFlag := flag.Bool("swagger", false, "Enable swagger endpoint")
	flag.Parse()

	// Determine the port to use
	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	// Determine the SQLite database file path
	sqlitePath := *sqliteFlag
	if sqlitePath == "" {
		sqlitePath = os.Getenv("SQLITE_PATH")
	}
	if sqlitePath == "" {
		sqlitePath = "webadmin.db"
	}

	// Determine the verbose mode
	verbose := *verboseFlag
	if !verbose {
		if os.Getenv("verbose") == "true" {
			verbose = true
		}
	}

	// Determine if metrics endpoint is enabled
	enableMetrics := *metricsFlag
	if !enableMetrics {
		if os.Getenv("metrics") == "true" {
			enableMetrics = true
		}
	}

	// Determine if swagger endpoint is enabled
	enableSwagger := *swaggerFlag
	if !enableSwagger {
		if os.Getenv("swagger") == "true" {
			enableSwagger = true
		}
	}
	return port, sqlitePath, verbose, enableMetrics, enableSwagger
}

// server the admin app
func serveAdminApp(port string, sqlitePath string, verbose bool, enableMetrics bool, enableSwagger bool) {
	// Initialize database
	if err := initDatabase(sqlitePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create a subdirectory file system for `dist/ng-matero/browser`
	subFS, err := fs.Sub(embeddedFiles, "dist/ng-matero/browser")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}

	// Initialize Fiber app
	config := fiber.Config{
		Prefork:               false,
		CaseSensitive:         true,
		StrictRouting:         false,
		ServerHeader:          "fiber",
		AppName:               "webadmin",
		DisableStartupMessage: true,
	}
	if verbose {
		config.DisableStartupMessage = false
	}
	app := fiber.New(config)

	// Add CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Add logger middleware
	if verbose {
		app.Use(logger.New(logger.Config{
			Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
		}))
	}

	// Register the metrics endpoint
	if enableMetrics {
		metricConfigs := monitor.Config{
			Title:   "Server Metrics",
			Refresh: 2 * time.Second,
		}
		app.Get("/metrics", monitor.New(metricConfigs))
	}

	// Register the swagger endpoint
	if enableSwagger {
		log.Println("Swagger endpoint enabled")
		app.Get("/swagger/*", swagger.HandlerDefault) // default

		app.Get("/swagger/*", swagger.New(swagger.Config{ // custom
			URL:         "http://localhost:8080/swagger/doc.json",
			DeepLinking: false,
			// Expand ("list") or Collapse ("none") tag groups by default
			DocExpansion: "none",
			// Prefill OAuth ClientId on Authorize popup
			OAuth: &swagger.OAuthConfig{
				AppName:  "OAuth Provider",
				ClientId: "21bb4edc-05a7-4afc-86f1-2e151e4ba6e2",
			},
			// Ability to change OAuth2 redirect uri location
			OAuth2RedirectUrl: "http://localhost:8080/swagger/oauth2-redirect.html",
		}))
	}

	// Setup API routes
	setupRoutes(app)

	// Serve files using Fiber's filesystem middleware
	app.Use("/", filesystem.New(filesystem.Config{
		Root:  http.FS(subFS),
		Index: "index.html",
	}))

	// Fallback route to serve index.html for unmatched routes
	app.Use(func(c *fiber.Ctx) error {
		// Read the embedded index.html file
		file, err := embeddedFiles.Open("dist/ng-matero/browser/index.html")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to load embedded index.html")
		}
		defer file.Close()

		// Serve the embedded index.html file
		return c.Type("html").SendStream(file)
	})

	log.Printf("Serving on http://127.0.0.1:%s\n", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func main() {
	serveAdminApp(parseFlags())
}
