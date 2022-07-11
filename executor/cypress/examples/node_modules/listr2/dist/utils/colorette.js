"use strict";
var _a;
Object.defineProperty(exports, "__esModule", { value: true });
const colorette = require("colorette");
/* istanbul ignore if */
if (((_a = process.env) === null || _a === void 0 ? void 0 : _a.LISTR_DISABLE_COLOR) === '1') {
    // disable coloring completely
    colorette.options.enabled = false;
}
exports.default = colorette;
