# Use as follows
# python3 testloggen.py > /var/tailtest/nohup.out

import time
import random
import string

if __name__ == "__main__":
    count = 1
    while(1):
        time.sleep(random.uniform(0.05, 2.0))
        letters = string.ascii_lowercase
        result_str = ''.join(random.choice(letters) for i in range(random.randint(0,9)))
        logline = str(count) + " RLine__" + result_str
        print(logline, flush=True)
        count += 1
