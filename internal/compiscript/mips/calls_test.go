package mips_test

import "testing"

func TestCallsInMARS(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"recursion", `function fib(n:integer):integer { if(n<=1) return n; else return fib(n-1)+fib(n-2); } print(fib(8));`, "21\n"},
		{"locals", `function sum(n:integer):integer { let saved=n; if(n==0) return 0; else { n=n-1; let child=sum(n); return saved+child+n; } } print(sum(4));`, "16\n"},
		{"mutual and globals", `let g=0; function even(n:integer):boolean {g=g+1;if(n==0)return true;else return odd(n-1);} function odd(n:integer):boolean {g=g+1;if(n==0)return false;else return even(n-1);} print(even(6)); print(g);`, "true\n7\n"},
		{"null return", `let g=0; function count(n:integer){if(n==0)return null;g=g+1;count(n-1);return;} count(5);print(g);`, "5\n"},
		{"float result", `function sum(n:integer):float {if(n==0)return 0;else return n+sum(n-1);}print(sum(4));`, "10\n"},
		{"pressure", `function add(a:integer,b:integer,c:integer,d:integer,e:integer,f:integer,g:integer,h:integer,i:integer,j:integer):integer{return a+b+c+d+e+f+g+h+i+j;}function one(n:integer):integer{return n;}print(add(one(1),one(2),one(3),one(4),one(5),one(6),one(7),one(8),one(9),one(10)));`, "55\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, stderr, code := execute(t, tt.source)
			if code != 0 || out != tt.want || stderr != "\n" {
				t.Fatalf("exit=%d stdout=%q stderr=%q want=%q", code, out, stderr, tt.want)
			}
		})
	}
}
