import inquirer from "inquirer";
import chalk from "chalk";
import logSymbols from "log-symbols";
import boxen from "boxen";
import { Strands } from "strands";

export async function askConfirmation(yes) {
  if (yes) {
    return;
  }

  const result = await inquirer.prompt([
    {
      type: "confirm",
      name: "response",
      message: "Do you want to continue?",
    },
  ]);

  if (result.response === false) {
    throw new Error("aborted");
  }
}

export const Screen = Strands;
export const log = console.log;

export const S = logSymbols;
export const C = chalk;
export const B = boxen;

export const started = (msg) => `${S.success} ${msg}`;

export const success = (msg) =>
  console.log(`
${C.bgGreen(C.black(" success "))} ${C.green(msg)}`);

export const failure = (msg) =>
  console.log(`
${C.bgRed(C.black(" failure "))} ${C.red(msg)}`);

export const precondition = (msg) =>
  console.log(`
${C.bgMagenta(C.black(" precondition "))} ${C.magenta(msg)}`);

export const warningInfo = (msg) =>
  console.log(`
${C.bgYellow(C.black(" info "))} ${C.yellow(msg)}`);

export const info = (msg) =>
  console.log(`
${C.bgBlueBright(C.white(" info "))} ${msg}`);
