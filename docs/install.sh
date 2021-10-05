echo "Getting kubectl-kubtest plugin"
#!/bin/sh 

if [ ! -z "${DEBUG}" ]; 
then set -x 
fi 

_detect_arch() { 
    case $(uname -m) in 
    amd64|x86_64) echo "x86_64" 
    ;; 
    arm64|aarch64) echo "arm64" 
    ;; 
    i386) echo "i386" 
    ;; 
    *) echo "Unsupported processor architecture"; 
    return 1 
    ;;
     esac 
}

_download_url() { 
        local arch="$(_detect_arch)" 
        if [ -z "$KUBTEST_VERSION" ]
        then
                local version=`curl -s https://api.github.com/repos/kubeshop/kubtest/releases/latest 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'`
                echo "https://github.com/kubeshop/kubtest/releases/download/${version}/kubtest_${version:1}_$(uname)_$arch.tar.gz"       
        else   
                echo "https://github.com/kubeshop/kubtest/releases/download/v${KUBTEST_VERSION}/kubtest_${KUBTEST_VERSION}_$(uname)_$arch.tar.gz" 
        fi
}

echo "Downloading kubtest from URL: $(_download_url)" 
curl -sSLf $(_download_url) > kubtest.tar.gz 
tar -xzf kubtest.tar.gz kubectl-kubtest 
rm kubtest.tar.gz
mv kubectl-kubtest /usr/local/bin/kubectl-kubtest
echo "kubectl-kubtest installed in /usr/local/bin/kubectl-kubtest"

