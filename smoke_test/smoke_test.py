import os
from pexpect.popen_spawn import PopenSpawn

print(os.getcwd())
os.chdir(os.getcwd() + "\check_rebuild")
print(os.getcwd())

child = PopenSpawn("air")
child.expect

a = child.expect("running", timeout=300)
if a == 0:
     with open("main.go", "a") as f:
        f.write("\n\n")
else:
    exit(0)

a = child.expect("running", timeout=300)
if a == 0:
    print("::set-output name=value::PASS")
else:
    print("::set-output name=value::FAIL")
    exit(0)
