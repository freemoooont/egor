/**
 * Re-export shadcn/ui-style components owned by this repo.
 *
 * Components were authored manually mirroring the canonical shadcn registry
 * (the CLI is interactive and we run unattended — see SCAFFOLD-NOTES.md).
 * Future `pnpm dlx shadcn@latest add <name>` runs should diff cleanly.
 */
export { Button, buttonVariants, type ButtonProps } from './button.tsx';
export { Input, type InputProps } from './input.tsx';
export { Label } from './label.tsx';
export {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from './card.tsx';
export { Alert, AlertTitle, AlertDescription } from './alert.tsx';
export { Logo } from './logo.tsx';
export { Avatar, AvatarImage, AvatarFallback } from './avatar.tsx';
export {
  Dialog,
  DialogTrigger,
  DialogPortal,
  DialogClose,
  DialogOverlay,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from './dialog.tsx';
export {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuGroup,
  DropdownMenuPortal,
  DropdownMenuCheckboxItem,
} from './dropdown-menu.tsx';
export { Separator } from './separator.tsx';
export { Skeleton } from './skeleton.tsx';
export { Switch } from './switch.tsx';
export { Progress } from './progress.tsx';
export { ScrollArea, ScrollBar } from './scroll-area.tsx';
export { Textarea, type TextareaProps } from './textarea.tsx';
