package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

//go:embed dist/ng-matero/browser/*
var embeddedFiles embed.FS

// parse command-line flags
func parseFlags() (string, bool, bool) {
	portFlag := flag.String("port", "", "Port to run the server on")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose mode")
	metricsFlag := flag.Bool("metrics", false, "Enable metrics endpoint")
	flag.Parse()

	// Determine the port to use
	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	// Determine the verbose mode
	verbose := *verboseFlag
	if !verbose {
		if os.Getenv("verbose") == "true" {
			verbose = true
		}
	}
	return port, verbose, *metricsFlag
}

// server the admin app
func serveAdminApp(port string, verbose bool, enableMetrics bool) {
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
	if verbose {
		app.Use(logger.New(logger.Config{
			Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
		}))
	}

	// Serve files using Fiber's filesystem middleware
	app.Use("/", filesystem.New(filesystem.Config{
		Root:  http.FS(subFS),
		Index: "index.html",
	}))

	// Register the metrics endpoint
	if enableMetrics {
		metricConfigs := monitor.Config{
			Title:   "Server Metrics",
			Refresh: 2 * time.Second,
		}
		app.Get("/metrics", monitor.New(metricConfigs))
	}

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
	port, verbose, enableMetrics := parseFlags()
	serveAdminApp(port, verbose, enableMetrics)
}
