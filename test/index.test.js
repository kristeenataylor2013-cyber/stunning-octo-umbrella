import assert from "node:assert/strict";
import test from "node:test";
import { greet } from "../src/index.js";

test("greets the world by default", () => {
  assert.equal(greet(), "Hello, world!");
});

test("greets a supplied name", () => {
  assert.equal(greet("Kristeena"), "Hello, Kristeena!");
});
