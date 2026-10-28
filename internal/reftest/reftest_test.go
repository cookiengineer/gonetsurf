package reftest

import "testing"

const reftestDocument = `<!doctype html><html><head><style>
html, body { margin:0; padding:0; background:#ffffff }
.box { position:absolute; left:10px; top:10px; width:40px; height:30px; background:#3366cc; border:3px solid #000000 }
</style></head><body><div class="box"></div></body></html>`

func TestIdenticalReftest(t *testing.T) {
	result, err := Compare([]byte(reftestDocument), []byte(reftestDocument), nil, Options{Width: 100, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	if result.Different != 0 {
		t.Errorf("identical documents differ: %s", result)
	}
}

func TestDifferentReftest(t *testing.T) {
	other := `<!doctype html><html><head><style>
html, body { margin:0; padding:0; background:#ffffff }
.box { position:absolute; left:10px; top:10px; width:40px; height:30px; background:#cc3333; border:3px solid #000000 }
</style></head><body><div class="box"></div></body></html>`
	result, err := Compare([]byte(reftestDocument), []byte(other), nil, Options{Width: 100, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	if result.Different == 0 {
		t.Fatal("different documents were reported identical")
	}
}
