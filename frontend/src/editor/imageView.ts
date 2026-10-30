/**
 * Image NodeViews. schema.ts' toDOM points at a `fb2://` URL nobody serves, so images resolve
 * href → `data:` URL here instead. `refresh` re-resolves every live view when the binaries change
 * (upload, rename, delete), which needs no PM remount and so keeps undo history.
 * Serialization is unaffected: pmDocToFB2 only reads node.attrs.href.
 */
import type { Node as PMNode } from "prosemirror-model";
import type { NodeView } from "prosemirror-view";
import type { Binary } from "../fb2/types";

interface ImageNodeView extends NodeView {
  refresh(): void;
}

export function createImageViews(getBinaries: () => Binary[]) {
  const views = new Set<ImageNodeView>();

  function resolve(href: string | undefined): string {
    if (!href) return "";
    const id = href.replace(/^#/, "");
    const bin = getBinaries().find((b) => b.ID === id);
    return bin ? `data:${bin.ContentType};base64,${bin.Data}` : "";
  }

  function create(node: PMNode): ImageNodeView {
    const wrap = document.createElement(node.type.name === "image_block" ? "div" : "span");
    wrap.className = "image";
    const img = document.createElement("img");
    wrap.appendChild(img);

    let current = node;
    const show = () => {
      const src = resolve(current.attrs.href);
      if (src) img.src = src;
      else img.removeAttribute("src");
      img.classList.toggle("missing", !src);
    };
    const render = () => {
      if (current.attrs.href) wrap.setAttribute("data-href", current.attrs.href);
      else wrap.removeAttribute("data-href");
      wrap.title = current.attrs.title || "";
      img.alt = current.attrs.alt || "";
      show();
    };
    render();

    const view: ImageNodeView = {
      dom: wrap,
      update(next) {
        if (next.type !== current.type) return false;
        current = next;
        render();
        return true;
      },
      refresh: show,
      destroy() {
        views.delete(view);
      },
    };
    views.add(view);
    return view;
  }

  return {
    nodeViews: { image_block: create, image_inline: create },
    refresh() {
      for (const v of views) v.refresh();
    },
  };
}
