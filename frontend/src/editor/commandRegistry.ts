/**
 * Every editor command by id. Hotkeys come from the `editor: true` rows of HOTKEY_ACTIONS, so a
 * row with no command here, or a typo in a caller, is a compile error rather than a dead button.
 */
import { undo, redo } from "prosemirror-history";
import type { Command } from "prosemirror-state";
import {
  toggleStrong, toggleEmphasis, toggleStrikethrough,
  toggleSub, toggleSup, toggleCode,
  styleNormal, styleSubtitle, styleTextAuthor,
  insertEmptyLine,
  cloneContainer, removeOuterContainer,
  addTitle, addEpigraph, addAnnotation, addTextAuthor,
  insertCite, insertPoem,
  mergeContainers,
} from "./commands";
import { findNext, findPrev } from "./search/plugin";
import type { EditorActionId } from "../settings/hotkeys";

export type CommandId = EditorActionId | "Undo" | "Redo";

export const EDITOR_COMMANDS: Record<CommandId, Command> = {
  Undo: undo,
  Redo: redo,
  FindNext: (_state, _dispatch, view) => (view ? findNext(view) : false),
  FindPrev: (_state, _dispatch, view) => (view ? findPrev(view) : false),
  ToggleStrong: toggleStrong,
  ToggleEmphasis: toggleEmphasis,
  ToggleStrikethrough: toggleStrikethrough,
  ToggleSub: toggleSub,
  ToggleSup: toggleSup,
  ToggleCode: toggleCode,
  StyleNormal: styleNormal,
  StyleSubtitle: styleSubtitle,
  StyleTextAuthor: styleTextAuthor,
  InsertEmptyLine: insertEmptyLine,
  CloneContainer: cloneContainer,
  RemoveOuterContainer: removeOuterContainer,
  AddTitle: addTitle,
  AddEpigraph: addEpigraph,
  AddAnnotation: addAnnotation,
  AddTextAuthor: addTextAuthor,
  InsertCite: insertCite,
  InsertPoem: insertPoem,
  MergeContainers: mergeContainers,
};
