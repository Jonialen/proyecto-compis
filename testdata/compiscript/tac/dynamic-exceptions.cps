function inc(x):integer { return x+1; }
let result=inc(2);
let effects=0;
try {
  effects=effects+1;
  print(result);
  let bad=1/0;
  effects=100;
} catch(error) {
  print(error);
  effects=effects+10;
}
print(effects);
