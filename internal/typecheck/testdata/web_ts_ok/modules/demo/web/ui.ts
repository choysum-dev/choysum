import styles from "./styles.css";
import icon from "./icon.svg";

export const label: string = "ok";
export const css: string = styles;
export const asset: string = icon;

export function greet(name: string): string {
  return `hello ${name}`;
}

void greet(label);
