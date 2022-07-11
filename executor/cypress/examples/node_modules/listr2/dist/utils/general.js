"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.cloneObject = void 0;
/**
 * Deep clones a object in the most easiest manner.
 */
function cloneObject(obj) {
    return JSON.parse(JSON.stringify(obj));
}
exports.cloneObject = cloneObject;
