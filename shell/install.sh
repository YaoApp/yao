 #!/bin/bash
VERSION="0.9.0"
URL="https://release.yaoapps.com/download"

function macos(){
    tmpdir=$(dirname $(mktemp -u))
    curl "$URL/yao-$VERSION-darwin-amd64" --output "$tmpdir/yao-$VERSION"
    ls -l $tmpdir/yao-$VERSION
    rmOldVersion
    sudo mv $tmpdir/yao-$VERSION /usr/local/bin/yao
    sudo chmod +x /usr/local/bin/yao
    echo ""
    echo /usr/local/bin/yao
    /usr/local/bin/yao version
    echo "DONE"
}

function linux(){
    tmpdir=$(dirname $(mktemp -u))
    curl "$URL/yao-$VERSION-linux-amd64" --output "$tmpdir/yao-$VERSION"
    ls -l $tmpdir/yao-$VERSION
    rmOldVersion
    sudo mv $tmpdir/yao-$VERSION /usr/local/bin/yao
    sudo chmod +x /usr/local/bin/yao
    echo ""
    echo /usr/local/bin/yao
    /usr/local/bin/yao version
    echo "DONE"
}


function rmOldVersion() {
    if command -v /usr/local/bin/yao &> /dev/null
    then
        sudo mv /usr/local/bin/yao /usr/local/bin/yao.bak
    fi
}

function rmOldVersionWin() {
    if command -v /usr/local/bin/yao &> /dev/null
    then
        mv /usr/local/bin/yao /usr/local/bin/yao.bak
    fi
}

function windows(){
    tmpdir=$(dirname $(mktemp -u))
    curl "$URL/yao-$VERSION-windows-386" --output "$tmpdir/yao-$VERSION"
    ls -l $tmpdir/yao-$VERSION
    rmOldVersionWin
    mv $tmpdir/yao-$VERSION /usr/local/bin/yao
    chmod +x /usr/local/bin/yao
    echo ""
    echo /usr/local/bin/yao
    /usr/local/bin/yao version
    echo "DONE"
}


if [ "$(uname)" = "Darwin" ];then
macos
elif [ "$(expr substr $(uname -s) 1 5)" = "Linux" ];then   
linux 
elif [ "$(expr substr $(uname -s) 1 9)" = "CYGWIN_NT" ];then    
windows
else 
echo $(uname -s) "does not support"
fi

