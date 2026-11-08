package mips_test

import (
	"testing"

	"genanalex/internal/compiscript/mips"
	"genanalex/internal/compiscript/tac"
)

func TestMIPSExceptions(t *testing.T) {
	for _, tt := range []struct {
		name, source, want string
		exit               int
	}{
		{"arithmetic and prefix", `print("before");try{let bad=1/0;print("wrong");}catch(e){print(e);}try{let bad=2%0;}catch(e){print(e);}print(7);`, "before\ndivision by zero\nremainder by zero\n7\n", 0},
		{"nested catches and identities", `try{let bad=1/0;}catch(first){let alias=first;print(first==alias);try{let bad=2%0;}catch(second){print(first==second);print(first);print(second);}print(first);}`, "true\nfalse\ndivision by zero\nremainder by zero\ndivision by zero\n", 0},
		{"catch fault selects outer", `let count=0;try{try{let bad=1/0;}catch(inner){count=1;let bad=2%0;count=100;}}catch(outer){print(outer);count=count+10;}print(count);`, "remainder by zero\n11\n", 0},
		{"recursive unwind and caller homes", `function fail(n:integer){if(n==0){let bad=1/0;}else{fail(n-1);}print("wrong");}function run(seed:integer):integer{let saved=seed+2;try{fail(5);}catch(e){print(e);}return seed+saved;}print(100+run(7));`, "division by zero\n116\n", 0},
		{"nearest recursive handler", `function run(n:integer){try{if(n==0){let bad=1/0;}else{run(n-1);print(n);return;}}catch(e){print(n);return;}}run(3);try{let bad=2%0;}catch(e){print(e);}`, "0\n1\n2\n3\nremainder by zero\n", 0},
		{"environment snapshot and abandoned captures", `function zero():integer{return 0;}let read=zero;function fail(){let n=2;function saved():integer{return n;}read=saved;n=3;let bad=1/0;}let n=9;try{let n=4;fail();}catch(e){print(n);}print(read());print(n);`, "9\n3\n9\n", 0},
		{"catch binding retained", `function zero():boolean{return false;}let read=zero;try{let bad=1/0;}catch(e){function same():boolean{print(e);return e==e;}read=same;}try{let bad=2%0;}catch(e){print(e);}print(read());`, "remainder by zero\ndivision by zero\ntrue\n", 0},
		{"normal and escaping cleanup", `function done(){try{print(1);return;}catch(e){print("wrong return");}}try{try{done();}catch(e){print("wrong normal");}while(true){try{break;}catch(e){print("wrong break");}}let n=0;while(n<3){n=n+1;try{continue;}catch(e){print("wrong continue");}}let bad=1/0;}catch(e){print(e);}done();`, "1\ndivision by zero\n1\n", 0},
		{"internal loop edges retain handler", `try{let n=0;while(n<2){n=n+1;continue;}while(true){break;}let bad=1/0;}catch(e){print(e);}`, "division by zero\n", 0},
		{"closure handlers are dynamic", `function zero():integer{return 0;}let run=zero;try{function fail():integer{let bad=1/0;return 0;}run=fail;}catch(e){print("wrong");}try{run();}catch(e){print(e);}`, "division by zero\n", 0},
		{"helper scratch and staged arguments unwind", `let text:string=null;try{print("a"+text);}catch(e){print(e);}function use(x:integer){print(x);}try{let x:integer;use(x);}catch(e){print(e);}use(3);`, "null string operand\nundefined binding\n3\n", 0},
		{"undefined binding fault", `try{let x:integer;print(x);}catch(e){print(e);}print(2);`, "undefined binding\n2\n", 0},
		{"uncaught fault", `print("before");let bad=1/0;`, "before\ndivision by zero\n", 1},
		{"uncaught catch fault", `try{let bad=1/0;}catch(e){print(e);let bad=2%0;}`, "division by zero\nremainder by zero\n", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, stderr, code := execute(t, tt.source)
			if out != tt.want || stderr != "\n" || code != tt.exit {
				t.Fatalf("stdout=%q stderr=%q exit=%d; want stdout=%q stderr=newline exit=%d", out, stderr, code, tt.want, tt.exit)
			}
		})
	}
}

func TestMIPSExceptionsInvalidIR(t *testing.T) {
	for _, tt := range []struct {
		name string
		tape []tac.Instruction
	}{
		{"missing catch label", []tac.Instruction{{Op: tac.PushHandler, Label: "missing"}}},
		{"unterminated handler", []tac.Instruction{{Op: tac.PushHandler, Label: "end"}, {Op: tac.Label, Label: "end"}}},
		{"pop underflow", []tac.Instruction{{Op: tac.PopHandler}}},
		{"branch pop underflow", []tac.Instruction{{Op: tac.BranchTrue, Label: "cleanup", Left: tac.Operand{Kind: tac.Literal, Type: "boolean", Value: "true"}}, {Op: tac.PushHandler, Label: "cleanup"}, {Op: tac.Label, Label: "cleanup"}, {Op: tac.PopHandler}}},
		{"catch without pending", []tac.Instruction{{Op: tac.CatchException, Target: tac.Operand{Kind: tac.Temporary, Value: "t0"}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assembly, err := mips.Generate(&tac.Program{Instructions: tt.tape})
			if err == nil || assembly != "" {
				t.Fatalf("want atomic infrastructure error; assembly=%q err=%v", assembly, err)
			}
		})
	}
}
