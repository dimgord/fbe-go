import { describe, it, expect, afterEach } from "vitest";
import { cleanPastedHTML, cleanPastedText, configurePaste, resetPasteConfigForTesting } from "./paste";

afterEach(() => resetPasteConfigForTesting());

describe("cleanPastedHTML", () => {
  it("strips Word conditional comments", () => {
    const src = `<p>hello</p><!--[if gte mso 9]><xml>junk</xml><![endif]--><p>world</p>`;
    expect(cleanPastedHTML(src)).toBe("<p>hello</p><p>world</p>");
  });

  it("removes <style> blocks", () => {
    const src = `<style>.x{color:red}</style><p>body</p>`;
    expect(cleanPastedHTML(src)).toBe("<p>body</p>");
  });

  it("strips mso- styles but keeps the paragraph", () => {
    const src = `<p style="mso-layout: auto; color: red;">text</p>`;
    expect(cleanPastedHTML(src)).toContain("<p>");
    expect(cleanPastedHTML(src)).toContain("text");
    expect(cleanPastedHTML(src)).not.toContain("mso");
    expect(cleanPastedHTML(src)).not.toContain("color");
  });

  it("drops class attributes", () => {
    const src = `<p class="MsoNormal">hi</p>`;
    expect(cleanPastedHTML(src)).toBe("<p>hi</p>");
  });

  it("drops <span> wrappers", () => {
    const src = `<p>a<span class="x">b</span>c</p>`;
    expect(cleanPastedHTML(src)).toBe("<p>abc</p>");
  });

  it("collapses multiple <br> into paragraph breaks", () => {
    const src = `<p>one<br><br><br>two</p>`;
    expect(cleanPastedHTML(src)).toBe("<p>one</p><p>two</p>");
  });

  it("drops empty paragraphs", () => {
    const src = `<p>x</p><p>   </p><p>y</p>`;
    expect(cleanPastedHTML(src)).toBe("<p>x</p><p>y</p>");
  });

  it("converts &nbsp; to regular space", () => {
    expect(cleanPastedHTML(`<p>a&nbsp;b</p>`)).toBe("<p>a b</p>");
  });

  it("preserves <strong> and <em> marks", () => {
    const src = `<p>hello <strong>bold</strong> and <em>italic</em></p>`;
    expect(cleanPastedHTML(src)).toBe("<p>hello <strong>bold</strong> and <em>italic</em></p>");
  });
});

describe("cleanPastedText", () => {
  it("normalizes CRLF / CR to LF", () => {
    expect(cleanPastedText("a\r\nb\rc")).toBe("a\nb\nc");
  });
  it("strips non-printable control chars (keeps tab/newline)", () => {
    expect(cleanPastedText("a\tb\ncd")).toBe("a\tb\ncd");
  });
  it("normalizes nbsp to space", () => {
    expect(cleanPastedText("a b")).toBe("a b");
  });
});

describe("configurePaste", () => {
  it("lets user pick a different replacement for NBSP (HTML path)", () => {
    configurePaste({ nbspChar: " " });
    expect(cleanPastedHTML("<p>a&nbsp;b</p>")).toBe("<p>a b</p>");
  });

  it("lets user pick a different replacement for NBSP (text path)", () => {
    configurePaste({ nbspChar: " " });
    expect(cleanPastedText("a b")).toBe("a b");
  });

  it("ignores non-single-character input", () => {
    configurePaste({ nbspChar: "  " });
    // Still the default regular space from resetPasteConfigForTesting().
    expect(cleanPastedHTML("<p>a&nbsp;b</p>")).toBe("<p>a b</p>");
  });
});
