#tinygo build -x -gc=leaking -target eosio -wasm-abi=generic -scheduler=none -opt 0 -tags=math_big_pure_go -gen-code=true -strip=false -o test.wasm .
mkdir -p build
tinygo build -x -gc=leaking -target eosio -wasm-abi=generic -scheduler=none -opt z -tags=math_big_pure_go -gen-code=true -strip=true -o build/test.wasm . || exit 1
