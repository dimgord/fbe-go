/** Path helpers for OS paths arriving from Go; either separator may appear. */
export function basename(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

export function dirname(path: string): string {
  return path.slice(0, path.length - basename(path).length).replace(/[\\/]$/, "");
}
