package routes_test

import (
	"github.com/gorilla/mux"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/armPelionEdge/devicedb/routes"

	"net/http"
    "net/http/httptest"
)

var _ = Describe("Kubernetes", func() {
	var router *mux.Router
    var kubernetesEndpoint *KubernetesEndpoint

    BeforeEach(func() {
        router = mux.NewRouter()
        kubernetesEndpoint = &KubernetesEndpoint{}
        kubernetesEndpoint.Attach(router)
	})
	
	Describe("/healthz", func() {
        Describe("GET", func() {
			It("Should respond with status code http.StatusOK", func() {
				req, err := http.NewRequest("GET", "/healthz", nil)

				Expect(err).Should(BeNil())

				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)

				Expect(rr.Code).Should(Equal(http.StatusOK))
			})
		})
	})
})
