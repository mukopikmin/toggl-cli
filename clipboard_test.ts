import { assertEquals, assertRejects } from "@std/assert";
import { ClipboardUnavailableError, writeClipboardText } from "./clipboard.ts";

Deno.test({
  name: "writeClipboardText hides unavailable command details",
  ignore: Deno.build.os !== "linux",
  async fn() {
    const originalPath = Deno.env.get("PATH");
    Deno.env.set("PATH", "");

    try {
      const error = await assertRejects(
        () => writeClipboardText("summary output"),
        ClipboardUnavailableError,
      );

      assertEquals(error.message, "Could not copy output to the clipboard.");
      assertEquals(error.message.includes("wl-copy"), false);
      assertEquals(error.message.includes("xclip"), false);
      assertEquals(error.message.includes("xsel"), false);
    } finally {
      if (originalPath === undefined) {
        Deno.env.delete("PATH");
      } else {
        Deno.env.set("PATH", originalPath);
      }
    }
  },
});
