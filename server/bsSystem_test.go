package verificat

/*
Integration Testing for Backstage
const (
	backstageURL = "https://backstage.rainbowq.co"
)
*/

/* DISABLED UNTIL BACKSTAGE WORKS

// TODO: A unit test version of this could use a fake BSSE object.
//
// Can we see match elements of each service entity?
func TestReadSystemBS(t *testing.T) {
	t.Run("Integration: reads Backstage catalog and matches system names", func(t *testing.T) {
		c, err := backstage.NewClient(backstageURL, "default", nil)
		assertError(t, err, nil)

		readTests := []struct {
			Name    string
			Service string
			Expect  string
			Client  *backstage.Client
		}{
			{"Admin", "admin", "code-owners-admin", c},
			{"Core", "core", "code-owners-core", c},
			{"AdServer", "ad-server", "code-owners-wasp", c},
		}

		for _, tt := range readTests {
			got, service, err := ReadSystemBS(tt.Service, c)
			want := tt.Expect
			assertError(t, err, nil)
			if diff := cmp.Diff(got, want); diff != "" {
				t.Error(diff)
				t.Errorf("For '%v' the service looks like\n: %v", tt.Service, service)
			}
		}
	})

	// Test that the SystemNotRecognized error is thrown
	t.Run("Integration: Handles only Systems it knows about", func(t *testing.T) {
		c, err := backstage.NewClient(backstageURL, "default", nil)
		assertError(t, err, nil)

		readTests := []struct {
			Name    string
			Service string
			Expect  string
			Client  *backstage.Client
		}{
			{"Core-App", "core-app", "", c},
			{"Prince", "Revolution", "", c},
			{"Rubidium-Strontium", "Isochron", "", c},
		}

		for _, tt := range readTests {
			got, service, err := ReadSystemBS(tt.Service, c)
			want := tt.Expect
			assertError(t, err, SystemNotRecognized)
			if diff := cmp.Diff(got, want); diff != "" {
				t.Error(diff)
				t.Errorf("For '%v' the service looks like\n: %v", tt.Service, service)
			}
		}
	})
}

// Integration: Backstage POST endpoint
func TestStoreIDs(t *testing.T) {
	store := StubServiceStore{
		map[string]int{},
		nil, nil,
	}
	server := NewVerificationServ(&store)

	// Use a real service - like /verificat/ - to check readiness with the Backstage API
	t.Run("Integration: Backstage returns 200 on POST", func(t *testing.T) {
		service := "verificat"
		request := newPostIDReq(service)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response.Code, http.StatusAccepted)

		if len(store.verifyCalls) != 1 {
			t.Errorf("got %d calls to TriggerID want %d", len(store.verifyCalls), 1)
		}

		if store.verifyCalls[0] != service {
			t.Errorf("did not store correct service got %q want %q", store.verifyCalls[0], service)
		}
	})
}

*/
