package mips_test

import "testing"

func TestClosureSignatureRejectionInMARS(t *testing.T) {
	out, stderr, code := execute(t, `function a(n:integer):integer{return n;} function b(n:boolean):integer{if(n)return 1;else return 0;} let alias=a;alias=b;print(alias(4));`)
	if code != 1 || out != "indirect call signature mismatch\n" || stderr != "\n" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, stderr)
	}
}
