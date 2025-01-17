import os
import sys
import json
import time
from inspect import currentframe, getframeinfo

test_dir = os.path.dirname(__file__)
sys.path.append(os.path.join(test_dir, '..'))

from ipyeos import log
from ipyeos.chaintester import ChainTester

logger = log.get_logger(__name__)

def get_line_number():
    cf = currentframe()
    return cf.f_back.f_lineno

def print_console(tx):
    cf = currentframe()
    filename = getframeinfo(cf).filename

    num = cf.f_back.f_lineno

    if 'processed' in tx:
        tx = tx['processed']
    for trace in tx['action_traces']:
        # logger.info(trace['console'])
        print(f'+++++console:{num}', trace['console'])

        if not 'inline_traces' in trace:
            continue
        for inline_trace in trace['inline_traces']:
            # logger.info(inline_trace['console'])
            print(f'+++++console:{num}', inline_trace['console'])

def print_except(tx):
    if 'processed' in tx:
        tx = tx['processed']
    logger.info(tx)
    for trace in tx['action_traces']:
        logger.info(trace['console'])
        logger.info(json.dumps(trace['except'], indent=4))

class Test(object):

    @classmethod
    def setup_class(cls):
        cls.main_token = 'UUOS'
        cls.chain = ChainTester()

        test_account1 = 'hello'
        a = {
            "account": test_account1,
            "permission": "active",
            "parent": "owner",
            "auth": {
                "threshold": 1,
                "keys": [
                    {
                        "key": 'EOS6AjF6hvF7GSuSd4sCgfPKq5uWaXvGM2aQtEUCwmEHygQaqxBSV',
                        "weight": 1
                    }
                ],
                "accounts": [{"permission":{"actor":test_account1,"permission": 'eosio.code'}, "weight":1}],
                "waits": []
            }
        }
        cls.chain.push_action('eosio', 'updateauth', a, {test_account1:'active'})
        cls.chain.push_action('eosio', 'setpriv', {'account':'hello', 'is_priv': True}, {'eosio':'active'})

    @classmethod
    def teardown_class(cls):
        cls.chain.free()

    def setup_method(self, method):
        pass

    def teardown_method(self, method):
        pass

    def test_ext(self):
        for wasm_file in ['test-cpp.wasm', 'test.wasm']:
            with open(wasm_file, 'rb') as f:
                code = f.read()
            with open('test.abi', 'r') as f:
                abi = f.read()
            self.chain.deploy_contract('hello', code, abi, 0)
            r = self.chain.push_action('hello', 'testext', {'a': 'hello', 'b':'aa'*32, 'c':'bb'*32})
            print_console(r)
            r = self.chain.push_action('hello', 'testext2', {'a': 'goodbye'})
            print_console(r)
            self.chain.produce_block()

            r = self.chain.push_action('hello', 'testopt', {'a': 'hello', 'b':'aa'*32, 'c':'bb'*32})
            print_console(r)
            # r = self.chain.pack_args('hello', 'testopt2', {'a': 'goodbye', 'b': None, 'c': None})
            # logger.info(r)
            r = self.chain.push_action('hello', 'testopt2', {'a': 'goodbye', 'b': None, 'c': None})
            print_console(r)
            self.chain.produce_block()

            r = self.chain.push_action('hello', 'testcombine', {'a': 'goodbye', 'b': 'aa'*32, 'c': 'cc'*32})
            print_console(r)
            self.chain.produce_block()
            self.chain.produce_block()

    def test_hello(self):
        with open('test.wasm', 'rb') as f:
            code = f.read()
        with open('test.abi', 'r') as f:
            abi = f.read()

        self.chain.deploy_contract('hello', code, abi, 0)
        try:
            r = self.chain.push_action('hello', 'sayhello', {'name': 'alice'})
            # r = self.chain.push_action('hello', 'sayhello', b'hello,world')
            print_console(r)
        except Exception as e:
            print_except(e.args[0])

        r = self.chain.push_action('hello', 'zzzzzzzzzzzzj', b'')

        r = self.chain.push_action('hello', 'testpointer', {'a': 'alice'})

    def test_math(self):
        with open('test.wasm', 'rb') as f:
            code = f.read()
        with open('test.abi', 'r') as f:
            abi = f.read()

        self.chain.deploy_contract('hello', code, abi, 0)
        try:
            r = self.chain.push_action('hello', 'testmath', b'')
            # r = self.chain.push_action('hello', 'sayhello', b'hello,world')
            print_console(r)
        except Exception as e:
            print_except(e.args[0])

    def test_variant(self):
        with open('test.wasm', 'rb') as f:
            code = f.read()
        with open('test.abi', 'r') as f:
            abi = f.read()

        self.chain.deploy_contract('hello', code, abi, 0)
        try:
            r = self.chain.push_action('hello', 'testvariant', b'')
            # r = self.chain.push_action('hello', 'sayhello', b'hello,world')
            print_console(r)
        except Exception as e:
            print_except(e.args[0])
