// package e2e
package e2e_test

import (
	"bufio"
	"bytes"
	"errors"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	
	"github.com/labstack/echo/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	// Import your app packages here (adjust paths to your project)
	"github.com/leomichalski/aigateway/internal/adapter/secrets"
	"github.com/leomichalski/aigateway/internal/port/outbound" 
	"github.com/leomichalski/aigateway/cmd" 
)

var _ = Describe("AI Gateway E2E API Tests with Testcontainers", Ordered, func() {
	const baseURL = "http://localhost:8081" // Use a different port for E2E tests
	const validToken = "Bearer sk_test_123"
	const invalidToken = "Bearer faketoken"
	const dbName = "gateway_e2e"

	var (
		ctx          context.Context
		cancel       context.CancelFunc
		mongoC       testcontainers.Container
		// redisC       testcontainers.Container
		mongoClient  *mongo.Client
		httpClient   *http.Client

		serverCtx    context.Context    // Dedicated context for Echo v5
		serverCancel context.CancelFunc // Used to shut down Echo
	)

	// =========================================================================
	// SETUP: Runs once before all tests
	// =========================================================================
	BeforeAll(func() {
		ctx, cancel = context.WithCancel(context.Background())
		httpClient = &http.Client{Timeout: 10 * time.Second}

		// Spin up MongoDB Container
		mongoReq := testcontainers.ContainerRequest{
			Image:        "mongo:6.0",
			ExposedPorts: []string{"27017/tcp"},
			Env: map[string]string{
				"MONGO_INITDB_ROOT_USERNAME": "root",
				"MONGO_INITDB_ROOT_PASSWORD": "password",
			},
			WaitingFor: wait.ForLog("Waiting for connections"),
		}
		var err error
		mongoC, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: mongoReq,
			Started:          true,
		})
		Expect(err).NotTo(HaveOccurred(), "Failed to start MongoDB container")

		mongoHost, _ := mongoC.Host(ctx)
		mongoPort, _ := mongoC.MappedPort(ctx, "27017")
		mongoURI := fmt.Sprintf("mongodb://root:password@%s:%s/%s?authSource=admin", mongoHost, mongoPort.Port(), dbName)

		// // Spin up Redis Container (For Rate Limiting Epic)
		// redisReq := testcontainers.ContainerRequest{
		// 	Image:        "redis:7-alpine",
		// 	ExposedPorts: []string{"6379/tcp"},
		// 	WaitingFor:   wait.ForLog("Ready to accept connections"),
		// }
		// redisC, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		// 	ContainerRequest: redisReq,
		// 	Started:          true,
		// })
		// Expect(err).NotTo(HaveOccurred(), "Failed to start Redis container")
		// // redisHost, _ := redisC.Host(ctx)
		// // redisPort, _ := redisC.MappedPort(ctx, "6379")
		// // (Set Redis URI in your EnvProvider here)

		// Connect to MongoDB
		mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		Expect(err).NotTo(HaveOccurred())
		Expect(mongoClient.Ping(ctx, nil)).To(Succeed())

		// =====================================================================
		// SETUP MONGODB INDEXES & MOCK DATA (Directly via mongo-driver)
		// =====================================================================
		
		db := mongoClient.Database(dbName)
		identitiesColl := db.Collection("identities")
		casbinColl := db.Collection("casbin_rule")

		// Configure Indexes (Enforce Data Integrity)
		_, err = identitiesColl.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: "api_key_hash", Value: 1}},
			Options: options.Index().SetUnique(true),
		})
		Expect(err).NotTo(HaveOccurred(), "Failed to create identities index")

		_, err = casbinColl.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "ptype", Value: 1},
				{Key: "v0", Value: 1},
				{Key: "v1", Value: 1},
				{Key: "v2", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		})
		Expect(err).NotTo(HaveOccurred(), "Failed to create casbin_rule index")

		// Seed AuthN Identities (Idempotent Upsert)
		// Note: Change the hash/tenant_id to match whatever your Ginkgo tests expect!
		// For the tests written above, you likely want tenant_id: "tenant-1"
		_, err = identitiesColl.UpdateOne(
			ctx,
			bson.M{"api_key_hash": "7ba88f380e2bd2b3a94f74a0f3cdb2b502fdf251f829ad44cd009de9f632ed96"},
			bson.M{
				"$set": bson.M{
					"tenant_id": "tenant-local-dev", 
					"tier":      "premium",
				},
			},
			options.Update().SetUpsert(true),
		)
		Expect(err).NotTo(HaveOccurred(), "Failed to seed identity")

		// Seed AuthZ Casbin Policies (Idempotent Upsert)
		devPolicies := []bson.M{
			{"ptype": "p", "v0": "tenant-local-dev", "v1": "gemini-3.1-pro-preview", "v2": "use"},
			{"ptype": "p", "v0": "tenant-local-dev", "v1": "gemini-3.5-flash-lite", "v2": "use"},
		}

		for _, policy := range devPolicies {
			_, err = casbinColl.UpdateOne(
				ctx,
				bson.M{
					"ptype": policy["ptype"],
					"v0":    policy["v0"],
					"v1":    policy["v1"],
					"v2":    policy["v2"],
				},
				bson.M{"$set": policy},
				options.Update().SetUpsert(true),
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to seed casbin policy")
		}
		// =====================================================================
		// INJECT DYNAMIC ENV VARS FOR EnvProvider
		// =====================================================================
		
		// Map of environment variables your EnvProvider expects
		envVars := map[string]string{
			"DATABASE_URI":   mongoURI,
			"DATABASE_NAME":  dbName,
			// "REDIS_URI":   redisURI,
			
			// Inject any LLM Provider keys needed by the ADK Adapter
			"GOOGLE_API_KEY": "fake-google-test-key",
			"DEBUG":          "true",
		}

		for key, val := range envVars {
			err := os.Setenv(key, val)
			Expect(err).NotTo(HaveOccurred(), "Failed to set env var: "+key)
			
			// Ginkgo v2 Magic: This schedules os.Unsetenv to run automatically 
			// at the very end of the suite, keeping your local dev machine clean!
			DeferCleanup(os.Unsetenv, key) 
		}
		envProvider := secrets.NewEnvProvider()
		// =====================================================================
		// Start the Echo v5 Server in a Goroutine
		// =====================================================================
		
		// Get an Echo instance
		e := cmd.BuildServer(mongoClient, outbound.DatabaseConfig{DBName: dbName}, envProvider)

		// Create a cancelable context specifically for the server
		serverCtx, serverCancel = context.WithCancel(context.Background())

		go func() {
			sc := echo.StartConfig{
				Address:         ":8081",
				GracefulTimeout: 2 * time.Second, // Shorter timeout for tests
			}
			
			// Start blocks until serverCtx is canceled
			if err := sc.Start(serverCtx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
				panic(fmt.Sprintf("Echo server failed: %v", err))
			}
		}()

		// Wait for the server to be healthy
		Eventually(func() int {
			resp, err := httpClient.Get(baseURL + "/healthz")
			if err != nil {
				return 0
			}
			defer resp.Body.Close()
			return resp.StatusCode
		}, "5s", "100ms").Should(Equal(http.StatusOK), "Echo server failed to start")


		// serverDone = make(chan struct{})
		// go func() {
		// 	defer close(serverDone)
		// 	// Start on 8081 so it doesn't conflict with local dev server
		// 	if err := e.Start(":8081"); err != nil && err != http.ErrServerClosed {
		// 		panic(err)
		// 	}
		// }()

		// // Wait for the server to be healthy
		// Eventually(func() int {
		// 	resp, err := httpClient.Get(baseURL + "/healthz")
		// 	if err != nil {
		// 		return 0
		// 	}
		// 	defer resp.Body.Close()
		// 	return resp.StatusCode
		// }, "5s", "100ms").Should(Equal(http.StatusOK), "Echo server failed to start")
	})

	// =========================================================================
	// TEARDOWN: Runs once after all tests finish
	// =========================================================================
	AfterAll(func() {
		// Shut down the server gracefully
		if serverCancel != nil {
			// Canceling this context triggers echo.StartConfig to gracefully
			// drain connections and shut down the HTTP server natively.
			serverCancel()
		}

		// Disconnect Mongo
		if mongoClient != nil {
			mongoClient.Disconnect(ctx)
		}

		// Terminate Containers
		if mongoC != nil {
			mongoC.Terminate(context.Background())
		}
		// if redisC != nil {
		// 	redisC.Terminate(context.Background())
		// }

		// Cancel the parent test context
		if cancel != nil {
			cancel()
		}
	})

	// =========================================================================
	// THE TESTS
	// =========================================================================
	
	// Helper function for sending chat requests
	// TODO: use generated client instead
	sendChatRequest := func(token string, reqBody map[string]interface{}) *http.Response {
		jsonData, err := json.Marshal(reqBody)
		Expect(err).NotTo(HaveOccurred())

		req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
		Expect(err).NotTo(HaveOccurred())

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", token)

		resp, err := httpClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
		return resp
	}

	Describe("POST /v1/chat/completions", func() {
		var basePayload map[string]interface{}

		BeforeEach(func() {
			// Base valid payload
			basePayload = map[string]interface{}{
				"messages": []map[string]string{
					{"role": "system", "content": "say hello"},
				},
				"temperature": 1,
			}
		})

		Context("Authentication & Authorization Gates", func() {
			It("should return 401 Unauthorized for fake token (AuthN Failure)", func() {
				basePayload["model"] = "gemini-3.5-flash-lite"
				basePayload["stream"] = false

				resp := sendChatRequest(invalidToken, basePayload)
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			})

			It("should return 403 Forbidden for unpermitted real model (AuthZ Failure)", func() {
				// Tenant-1 is NOT seeded with this policy
				basePayload["model"] = "gemini-3.5-flash"
				basePayload["stream"] = false

				resp := sendChatRequest(validToken, basePayload)
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			})

			It("should return 403 Forbidden for fake/unregistered model (AuthZ Failure)", func() {
				basePayload["model"] = "dsjandjansj"
				basePayload["stream"] = false

				resp := sendChatRequest(validToken, basePayload)
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			})
		})

		Context("Successful Generations", func() {
			It("should return 200 OK for allowed model with stream=false", func() {
				basePayload["model"] = "gemini-3.5-flash-lite" // Tenant-1 IS seeded with this policy
				basePayload["stream"] = false

				resp := sendChatRequest(validToken, basePayload)
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("application/json"))
			})

			It("should return 200 OK and stream Server-Sent Events when stream=true", func() {
				basePayload["model"] = "gemini-3.5-flash-lite"
				basePayload["stream"] = true
				basePayload["messages"].([]map[string]string)[0]["content"] = "count to 2"

				resp := sendChatRequest(validToken, basePayload)
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("text/event-stream"))

				// Validate SSE Data chunking
				reader := bufio.NewReader(resp.Body)
				foundDataChunk := false
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "data: ") {
						foundDataChunk = true
						break
					}
				}
				Expect(foundDataChunk).To(BeTrue(), "Expected to receive 'data: ' chunks")
			})
		})
	})
})