package persist

import (
	"testing"
)

func TestBlobHandler(t *testing.T) {
	////reg := New(
	////	WithBlobService(NewBlobService(local.NewFS(".tmp/"))),
	////	WithReferrersSupport(true),
	////)
	//
	//srv := httptest.NewServer(reg)
	//defer srv.Close()
	//
	//ref, err := name.ParseReference(fmt.Sprintf("%s/test/bar:latest", strings.TrimPrefix(srv.URL, "http://")))
	//if err != nil {
	//	t.Fatal(err)
	//}
	//img, err := random.Image(1024, 5)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//if err := remote.Write(ref, img); err != nil {
	//	t.Fatalf("remote.Write: %v", err)
	//}
}
