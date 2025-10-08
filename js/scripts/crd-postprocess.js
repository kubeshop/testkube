const fs = require("fs");
const buf = require("buffer");

// This script should be run after controller-gen
// This script copies all CRD files found in `src` and puts them into `dst`.
// All files will be pre- and postfixed.
const src = "k8s/crd";
const dst = "k8s/helm/testkube-operator/templates";
const prefix = "{{- if .Values.installCRD }}\n";
const postfix = "{{- end }}\n";

fs.readdir(src, (err, files) => {
  files.forEach((file) => {
    const data = fs.readFileSync(src + "/" + file);
    const fd = fs.openSync(dst + "/" + file, "w+");
    const prefixBuffer = buf.Buffer.from(prefix);
    const postfixBuffer = buf.Buffer.from(postfix);
    fs.writeSync(fd, prefixBuffer, 0, prefixBuffer.length, 0);
    fs.writeSync(fd, data, 0, data.length, prefixBuffer.length);
    fs.writeSync(
      fd,
      postfixBuffer,
      0,
      postfixBuffer.length,
      data.length + prefixBuffer.length
    );
    fs.close(fd, (err) => {
      if (err) throw err;
    });
  });
});
