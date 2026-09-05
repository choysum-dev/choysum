import "./styles.css";
import icon from "./icon.svg";

export const label: string = "ok";
export const asset: string = icon;

export function greet(name: string): string {
  return `hello ${name}`;
}

void greet(label);
