package main

import (
	"net/http"
	"os"

	_ "github.com/Agora-Calyce/spx-tf-api/docs"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/handlers"
	"github.com/Agora-Calyce/spx-tf-api/internal/api/store"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title SPX TF Proxy API
// @version 1.0
// @description Proxy API used to build a Terraform provider.
// @host localhost:8080
// @BasePath /

func main() {
	// zerolog configuration
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Initialize Echo
	e := echo.New()

	// Initialize the stores
	configStore := store.NewConfigStore()

	// Initialize the default configuration from an environment variable
	defaultEndpoint := os.Getenv("SPX_API_ENDPOINT")
	if defaultEndpoint == "" {
		log.Fatal().Msg("The target API URL must be set through the SPX_API_ENDPOINT environment variable")
	}
	configStore.SetEndpoint(defaultEndpoint)
	log.Info().Str("endpoint", defaultEndpoint).Msg("Endpoint initialized")

	// Initialize the handlers
	vpcHandler := handlers.NewVPCHandler(configStore)
	subnetHandler := handlers.NewSubnetHandler(configStore)
	diskHandler := handlers.NewDiskHandler(configStore)
	eipHandler := handlers.NewEipHandler(configStore)
	lbHandler := handlers.NewLoadBalancerHandler(configStore)
	sshKeyHandler := handlers.NewSSHKeyHandler(configStore)
	instanceHandler := handlers.NewInstanceHandler(configStore)
	sgHandler := handlers.NewSecurityGroupHandler(configStore)

	// Base middlewares
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Health route
	// @Summary Check the API status
	// @Description Returns the operational status of the API.
	// @Tags health
	// @Produce json
	// @Success 200 {object} map[string]string
	// @Router /health [get]
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "up",
		})
	})

	// Swagger UI
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// VPC routes
	e.POST("/:orga_id/:az/:project_id/vpcs", vpcHandler.CreateVPC)
	e.GET("/:orga_id/:az/:project_id/vpcs", vpcHandler.ListVPCs)
	e.GET("/:orga_id/:az/:project_id/vpcs/:id", vpcHandler.GetVPC)
	e.PUT("/:orga_id/:az/:project_id/vpcs/:id", vpcHandler.UpdateVPC)
	e.DELETE("/:orga_id/:az/:project_id/vpcs/:id", vpcHandler.DeleteVPC)

	// Subnet routes
	e.POST("/:orga_id/:az/:project_id/subnets", subnetHandler.CreateSubnet)
	e.GET("/:orga_id/:az/:project_id/subnets", subnetHandler.ListSubnets)
	e.GET("/:orga_id/:az/:project_id/subnets/:id", subnetHandler.GetSubnet)
	e.PUT("/:orga_id/:az/:project_id/subnets/:id", subnetHandler.UpdateSubnet)
	e.DELETE("/:orga_id/:az/:project_id/subnets/:id", subnetHandler.DeleteSubnet)

	// Disk routes
	e.POST("/:orga_id/:az/:project_id/disks", diskHandler.CreateDisk)
	e.GET("/:orga_id/:az/:project_id/disks", diskHandler.ListDisks)
	e.GET("/:orga_id/:az/:project_id/disks/:id", diskHandler.GetDisk)
	e.PUT("/:orga_id/:az/:project_id/disks/:id", diskHandler.UpdateDisk)
	e.DELETE("/:orga_id/:az/:project_id/disks/:id", diskHandler.DeleteDisk)
	e.GET("/:orga_id/:az/:project_id/storage-classes", diskHandler.ListStorageClasses)

	// EIP routes
	e.POST("/:orga_id/:az/:project_id/eips", eipHandler.CreateEip)
	e.GET("/:orga_id/:az/:project_id/eips", eipHandler.ListEips)
	e.GET("/:orga_id/:az/:project_id/eips/:id", eipHandler.GetEip)
	e.PUT("/:orga_id/:az/:project_id/eips/:id", eipHandler.UpdateEip)
	e.DELETE("/:orga_id/:az/:project_id/eips/:id", eipHandler.DeleteEip)

	// Load Balancer routes
	e.POST("/:orga_id/:az/:project_id/load-balancers", lbHandler.CreateLoadBalancer)
	e.GET("/:orga_id/:az/:project_id/load-balancers", lbHandler.ListLoadBalancers)
	e.GET("/:orga_id/:az/:project_id/load-balancers/:id", lbHandler.GetLoadBalancer)
	e.PUT("/:orga_id/:az/:project_id/load-balancers/:id", lbHandler.UpdateLoadBalancer)
	e.DELETE("/:orga_id/:az/:project_id/load-balancers/:id", lbHandler.DeleteLoadBalancer)

	// SSH key routes
	e.POST("/:orga_id/:az/:project_id/ssh-keys", sshKeyHandler.CreateSSHKey)
	e.GET("/:orga_id/:az/:project_id/ssh-keys", sshKeyHandler.ListSSHKeys)
	e.GET("/:orga_id/:az/:project_id/ssh-keys/:id", sshKeyHandler.GetSSHKey)
	e.DELETE("/:orga_id/:az/:project_id/ssh-keys/:id", sshKeyHandler.DeleteSSHKey)

	// Instance routes
	e.POST("/:orga_id/:az/:project_id/instances", instanceHandler.CreateInstance)
	e.GET("/:orga_id/:az/:project_id/instances", instanceHandler.ListInstances)
	e.GET("/:orga_id/:az/:project_id/instances/:id", instanceHandler.GetInstance)
	e.PUT("/:orga_id/:az/:project_id/instances/:id", instanceHandler.UpdateInstance)
	e.DELETE("/:orga_id/:az/:project_id/instances/:id", instanceHandler.DeleteInstance)
	e.GET("/:orga_id/:az/:project_id/vm-types", instanceHandler.ListVMTypes)

	// Security group routes
	e.POST("/:orga_id/:az/:project_id/security-groups", sgHandler.CreateSecurityGroup)
	e.GET("/:orga_id/:az/:project_id/security-groups", sgHandler.ListSecurityGroups)
	e.GET("/:orga_id/:az/:project_id/security-groups/:eid", sgHandler.GetSecurityGroup)
	e.POST("/:orga_id/:az/:project_id/security-groups/:eid", sgHandler.UpdateSecurityGroup)
	e.DELETE("/:orga_id/:az/:project_id/security-groups/:eid", sgHandler.DeleteSecurityGroup)

	// TODO: Configure the proxy to the target API once the swagger is received
	// e.Any("/proxy/*", handlers.ProxyHandler)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info().Str("port", port).Msg("Starting server")
	if err := e.Start(":" + port); err != nil {
		log.Fatal().Err(err).Msg("Error starting server")
	}
}
