#!/usr/bin/env python3
#
# ZAP is an HTTP/HTTPS proxy for assessing web application security.
#
# This script emulates the official ZAP script for testing purposes

import getopt
import logging
import sys

logging.basicConfig(level=logging.INFO, format='%(asctime)s %(message)s')

def main(argv):
    target = ''
    ignore_warn = False

    pass_count = 0
    warn_count = 0
    fail_count = 0

    try:
        opts, args = getopt.getopt(argv, "t:c:u:g:m:n:r:J:w:x:l:hdaijp:sz:P:D:T:IU:", ["hook="])
    except getopt.GetoptError as exc:
        logging.warning('Invalid option ' + exc.opt + ' : ' + exc.msg)
        sys.exit(3)

    for opt, arg in opts:
        if opt == '-t':
            target = arg
        elif opt == '-d':
            logging.getLogger().setLevel(logging.DEBUG)
        elif opt == '-I':
            ignore_warn = True

    print('2022-03-30 07:36:08,706 Could not find custom hooks file at /home/zap/.zap_hooks.py')
    print('Using the Automation Framework')
    print('Downloading add-on from: https://github.com/zaproxy/zap-extensions/releases/download/pscanrulesBeta-v28/pscanrulesBeta-beta-28.zap')
    print('Add-on downloaded to: /home/zap/.ZAP/plugin/pscanrulesBeta-beta-28.zap')
    print('Total of 615 URLs')

    if target == 'https://www.example.com/pass/':
        print('PASS: Vulnerable JS Library [10003]')
        print('PASS: Cookie No HttpOnly Flag [10010]')
        pass_count = 2
    elif target == 'https://www.example.com/warn/':
        print('PASS: X-AspNet-Version Response Header [10061]')
        print('WARN-NEW: Re-examine Cache-control Directives [10015] x 12 ')
        pass_count = 1
        warn_count = 1
    elif target == 'https://www.example.com/fail/':
        print('WARN-NEW: Re-examine Cache-control Directives [10015] x 12 ')
        print('        https://www.example.com (200 OK)')
        print('FAIL: Unknown issue')
        print('        https://www.example.com (200 OK)')
        pass_count = 0
        warn_count = 1
        fail_count = 1

    if fail_count > 0:
        sys.exit(1)
    elif (not ignore_warn) and warn_count > 0:
        sys.exit(2)
    elif pass_count > 0:
        sys.exit(0)
    else:
        sys.exit(3)

if __name__ == "__main__":
    main(sys.argv[1:])
