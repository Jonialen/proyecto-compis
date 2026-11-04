package mips_test

import "testing"

func TestClosuresInMARS(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"escaping counters", `function zero():integer{return 0;} let saved=zero; function install(seed:integer){let count=seed; function next():integer{count=count+1;return count;} saved=next;} install(10); let a=saved; install(20); let b=saved; print(a());print(b());print(a());`, "11\n21\n12\n"},
		{"function identity", `function f():integer{return 1;} function same(x,y):boolean{return x==y;} function different(x,y):boolean{return x!=y;} print(same(f,f));print(different(f,f));`, "true\nfalse\n"},
		{"shared capture", `function zero():integer{return 0;} let inc=zero;let read=zero;function install(){let n=3;function next():integer{n=n+1;return n;}function get():integer{return n;}inc=next;read=get;}install();print(inc());print(read());`, "4\n4\n"},
		{"block reentry", `function zero():integer{return 0;}let a=zero;let b=zero;let i=0;while(i<2){let n=i;function read():integer{return n;}if(i==0)a=read;else b=read;i=i+1;}print(a());print(b());`, "0\n1\n"},
		{"recursive owner", `function outer(n:integer):integer{let seed=n;function read():integer{return seed;}if(n==0)return read();else return read()+outer(n-1)+read();}print(outer(4));`, "20\n"},
		{"callee selection", `function first(n:integer):integer{return n+10;}function second(n:integer):integer{return n+20;}let alias=first;function change():integer{alias=second;return 1;}print(alias(change()));print(alias(2));`, "11\n22\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, stderr, code := execute(t, tt.source)
			if code != 0 || out != tt.want || stderr != "\n" {
				t.Fatalf("exit=%d stdout=%q stderr=%q want=%q", code, out, stderr, tt.want)
			}
		})
	}
}
