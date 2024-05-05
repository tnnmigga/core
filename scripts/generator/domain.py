import os
import sys

path = ""
name = ""
pkg = ""

def readFile(name):
    with open(name, "r") as f:
        return f.read()


def writeFile(name, text):
    if os.path.exists(name):
        os.remove(name)
    with open(name, "w") as f:
        f.write(text)
    os.system("go fmt " + name)


def genUseCase():
    global path, name
    text = '''
    package %s

    import (
        "%s/%s/domain"
        "%s/%s/domain/api"

        "github.com/tnnmigga/core/idef"
    )

    type useCase struct {
        *domain.Domain
    }

    func New(d *domain.Domain) api.I%s {
        c := &useCase{
            Domain: d,
        }
        c.After(idef.ServerStateInit, c.afterInit)
        return c
    }

    func (c *useCase) afterInit() error {
        return nil
    }

    ''' % (name.lower(), pkg, path, pkg, path, name)
    dirname = path + "/domain/impl/" + name.lower()
    writeFile(dirname + "/usecase.go", text)

def genApi():
    global path, name
    text = '''
    package api

    type I%s interface {
    }

    ''' % name
    writeFile(path + "/domain/api/" + name.lower() + ".go", text)

def genDomain():
    global path, name
    text = readFile(path + "/domain/domain.go")
    index = text.find("MaxCaseIndex", 300, len(text))
    text = text[:index] + name + "CaseIndex\n" + text[index:]
    writeFile(path + "/domain/domain.go", text)

def genImpl():
    global path, name
    text = readFile(path + "/domain/impl/impl.go").strip()
    text = text[:-1] + "d.PutCase(domain.{}CaseIndex, {}.New(d))\n".format(name, name.lower()) + text[-1:]
    index = text.find(")")
    text = text[:index] + "\"{}/{}/domain/impl/{}\"\n".format(pkg, path, name.lower()) + text[index:]
    writeFile(path + "/domain/impl/impl.go", text)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("cmd argv error")
        exit()
    for arg in sys.argv[1:]:
        key, value = arg.split("=")
        if key == "name":
            name = value
        if key == "path":
            path = value
            if path[-1] == "/":
                path = path[:-1]
    dirname = path + "/domain/impl/" + name.lower()
    mod = readFile("./go.mod")
    mod = mod.strip()
    pkg = mod.split("\n")[0].split(" ")[1].strip()
    if os.path.exists(dirname):
        print("useCase already exists")
        exit()
    os.mkdir(dirname)
    genUseCase()
    genDomain()
    genApi()
    genImpl()
    print(name, "case generated")