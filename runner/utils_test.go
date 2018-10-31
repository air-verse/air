package runner

import "testing"

func Test_CmdGen(t *testing.T) {

	fk := Engine{}
	prms := fk.cmdgen("ping", []string{"-c", "10", "4.2.2.1"})

	if len(prms) != 5 {
		t.Errorf("not expected lenght: %d", len(prms))
	}

}
