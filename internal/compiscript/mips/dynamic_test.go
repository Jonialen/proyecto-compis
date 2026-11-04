package mips_test

import (
	"testing"

	"genanalex/internal/compiscript"
	"genanalex/internal/compiscript/mips"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

func TestMIPSDynamic(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"tagged operations", `function add(x,y){print(x+y);print(x==y);}add("a\u0000","á");add(1,2.5);function work(x,flag):float{print(!flag);print(x<3);return -x/2;}print(work(2,false));`, "a\x00á\nfalse\n3.5\nfalse\ntrue\ntrue\n-1\n"},
		{"dynamic members", `class A{let n:integer=1;function add(x):integer{n=n+x;return n;}}function update(x):integer{x.n=3;return x.add(2);}let box=new A();print(update(box));print(box.n);`, "5\n5\n"},
		{"generic identity and null", `let saved;function keep(xs):integer[]{saved=xs;return xs;}let typed=keep([]);print(saved==typed);function nil(x){print(x);}let text:string=null;nil(text);`, "true\nnull\n"},
		{"float check does not mutate alias", `function integerOnly(x:integer){print(x);}function floatOnly(x:float){print(x);}function consume(f,x){f(x);}let n=2;consume(floatOnly,n);integerOnly(n);`, "2\n2\n"},
		{"runtime numeric result tags", `function integerOnly(x:integer){print(x);}function floatOnly(x:float){print(x);}function calc(x,y){integerOnly(x+y);integerOnly(x-y);integerOnly(x*y);integerOnly(x%y);floatOnly(x/y);}calc(7,2);function same(x,y):boolean{return x==y;}print(same(2,2.0));print(same(null,[1]));`, "9\n5\n14\n1\n3.5\ntrue\nfalse\n"},
		{"unknown field changes actual tags", `class Box{let value=null;}function put(box,value){box.value=value;}let box=new Box();put(box,[1]);print(box.value);put(box,"á");print(box.value);`, "[1]\ná\n"},
		{"nested checked list covariance", `class A{let n:integer=3;}class B:A{}function read(xs:A[][]):integer{return xs[0][0].n;}function pass(xs):integer{return read(xs);}print(pass([[new B()]]));function count(xs:integer[]):integer{let n=0;foreach(v in xs){n=n+1;}return n;}function empty(xs):integer{return count(xs);}print(empty([]));`, "3\n0\n"},
		{"generic mutation", `function put(xs,value){xs[0]=value;}let xs=[1];put(xs,4);print(xs);`, "[4]\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertReferenceExecution(t, tt.source, tt.want, 0)
		})
	}
}

func TestMIPSDynamicFaults(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"invalid negation", `function work(x){print(!x);}work(1);`, "invalid boolean negation\n"},
		{"typed list shape", `function read(xs:integer[][]){print("wrong");}function pass(xs){read(xs);}pass([[true]]);`, "list type mismatch\n"},
		{"covariant view checks actual element", `class A{}class B:A{let n:integer=3;}let b=[new B()];let a:A[]=b;a[0]=new A();print(b[0].n);`, "nominal object type mismatch\n"},
		{"dynamic assignment returns converted box", `class A{let n:float=0;}function integerOnly(x:integer){print("wrong");}function set(box){let value=(box.n=1);print(box.n);integerOnly(value);}set(new A());`, "1\ndynamic value type mismatch\n"},
		{"null list", `let xs:integer[]=null;function read(xs){print(xs[0]);}read(xs);`, "null list\n"},
		{"incompatible equality", `function same(x,y){print(x==y);}same(1,true);`, "invalid binary operand types\n"},
		{"nominal parameter", `class A{}class B{}function body(x:A){print("wrong");}function pass(x){body(x);}pass(new B());`, "nominal object type mismatch\n"},
		{"allocation contract", `function put(xs,x){xs[0]=x;}put([1],true);`, "dynamic value type mismatch\n"},
		{"fractional integer index", `let a=[1];let i=3/2;function read(xs,n){print(xs[n]);}read(a,i);`, "invalid list index\n"},
		{"nonfinite integer index", `let n=1;let i=0;while(i<400){n=n*10;i=i+1;}function read(xs,k){print(xs[k]);}read([1],n);`, "invalid list index\n"},
		{"huge index", `function read(xs,n){print(xs[n]);}read([1],1000000000000);`, "list index out of bounds\n"},
		{"readonly", `class A{const n:integer=1;}function put(x){x.n=2;}put(new A());`, "immutable field\n"},
		{"missing", `class A{}function read(x){print(x.missing);}read(new A());`, "missing object member missing\n"},
		{"undefined", `class A{let n:integer;}print(new A().n);`, "undefined field s2.n\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertReferenceExecution(t, tt.source, tt.want, 1)
		})
	}
}

func TestMIPSDynamicMalformedStaticOperation(t *testing.T) {
	r := compiscript.Compile([]byte(`print(1+2);`))
	if r.TAC == nil {
		t.Fatal(r.Diagnostics)
	}
	for n := range r.TAC.Instructions {
		i := &r.TAC.Instructions[n]
		if i.Op == tac.Add {
			i.Right.Type = model.TypeBoolean
			i.Right.Value = "true"
		}
	}
	assembly, err := mips.Generate(r.TAC)
	if assembly != "" || err == nil {
		t.Fatalf("invalid static contract: assembly bytes=%d err=%v", len(assembly), err)
	}
}
