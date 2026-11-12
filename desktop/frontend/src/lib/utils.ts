import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

/**
 * dnd-kit listeners without the keyboard one, for making a whole row a pointer drag
 * handle: Space and Enter stay free for the app's own shortcuts (the grip keeps the
 * keyboard drag).
 */
export function pointerOnly<T extends { onKeyDown?: unknown }>(listeners: T | undefined): Omit<T, "onKeyDown"> | undefined {
  if (!listeners) return undefined;
  const { onKeyDown: _ignored, ...rest } = listeners;
  void _ignored;
  return rest;
}
