/**
 * Reusable Tailwind class combinations for the Elegant Design System
 * These are meant to be used with template literals and cn() utility
 */

// Button Styles
export const buttonBase = "inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-[0.625rem] font-medium text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2";

export const buttonPrimary = `${buttonBase} bg-accent-primary text-white shadow-[0_1px_3px_0_rgba(0,0,0,0.04),0_1px_2px_0_rgba(0,0,0,0.03)] hover:bg-accent-primary/90 hover:shadow-[0_4px_6px_-1px_rgba(0,0,0,0.05),0_2px_4px_-1px_rgba(0,0,0,0.03)] focus:ring-accent-primary/50 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed`;

export const buttonSecondary = `${buttonBase} bg-elegant-100 text-elegant-900 dark:bg-dark-surface dark:text-dark-text shadow-[0_1px_2px_0_rgba(0,0,0,0.02)] hover:bg-elegant-200 dark:hover:bg-dark-border hover:shadow-[0_1px_3px_0_rgba(0,0,0,0.04),0_1px_2px_0_rgba(0,0,0,0.03)] focus:ring-elegant-300 active:scale-[0.98]`;

export const buttonGhost = `${buttonBase} bg-transparent text-elegant-700 dark:text-dark-text hover:bg-elegant-100 dark:hover:bg-dark-surface focus:ring-elegant-300 active:scale-[0.98]`;

export const buttonIcon = "p-2 rounded-[0.625rem] transition-all duration-200 hover:bg-elegant-100 dark:hover:bg-dark-surface active:scale-95 focus:outline-none focus:ring-2 focus:ring-elegant-300";

// Card Styles
export const cardBase = "rounded-2xl bg-white dark:bg-dark-elevated shadow-[0_1px_3px_0_rgba(0,0,0,0.04),0_1px_2px_0_rgba(0,0,0,0.03)] border border-elegant-200/50 dark:border-dark-border transition-all duration-200";

export const cardHover = `${cardBase} hover:shadow-[0_10px_15px_-3px_rgba(0,0,0,0.06),0_4px_6px_-2px_rgba(0,0,0,0.03)] hover:border-elegant-300/50 dark:hover:border-elegant-800`;

// Input Styles
export const inputBase = "w-full px-4 py-2.5 rounded-[0.625rem] bg-white dark:bg-dark-surface border border-elegant-300 dark:border-dark-border text-elegant-900 dark:text-dark-text placeholder-elegant-400 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-accent-primary/20 focus:border-accent-primary";

export const textareaBase = `${inputBase} resize-none`;

// Badge Styles
export const badgeBase = "inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium transition-all duration-200";

export const badgePrimary = `${badgeBase} bg-accent-primary/10 text-accent-primary`;
export const badgeSuccess = `${badgeBase} bg-accent-success/10 text-accent-success`;
export const badgeWarning = `${badgeBase} bg-accent-warning/10 text-accent-warning`;
export const badgeError = `${badgeBase} bg-accent-error/10 text-accent-error`;
export const badgeNeutral = `${badgeBase} bg-elegant-100 dark:bg-dark-surface text-elegant-700 dark:text-dark-text`;

// Loading Styles
export const loadingSpinner = "inline-block h-5 w-5 animate-spin rounded-full border-2 border-elegant-300 border-t-accent-primary";

// Timeline Styles
export const timelineContainer = "relative pl-8 before:content-[''] before:absolute before:left-0 before:top-0 before:bottom-0 before:w-px before:bg-elegant-200 dark:before:bg-dark-border";

export const timelineDot = "absolute -left-8 top-1 w-4 h-4 rounded-full border-2 border-white dark:border-dark-elevated bg-accent-primary shadow-[0_1px_3px_0_rgba(0,0,0,0.04),0_1px_2px_0_rgba(0,0,0,0.03)]";

export const timelineDotActive = `${timelineDot} animate-pulse`;

// Glass Morphism
export const glassEffect = "backdrop-blur-lg bg-white/70 dark:bg-dark-elevated/70 border border-white/20 dark:border-dark-border/20";

// Utility Classes
export const textGradient = "bg-clip-text text-transparent bg-gradient-to-r from-accent-primary to-accent-secondary";

export const hoverLift = "transition-transform duration-200 hover:-translate-y-0.5";

export const divider = "h-px bg-elegant-200 dark:bg-dark-border my-6";

// Code Block
export const codeBlock = "rounded-2xl bg-elegant-900 dark:bg-dark-surface p-6 overflow-x-auto font-mono text-sm leading-relaxed text-elegant-100 shadow-[0_20px_25px_-5px_rgba(0,0,0,0.07),0_10px_10px_-5px_rgba(0,0,0,0.02)]";

/**
 * Utility function to conditionally join class names
 * Similar to clsx or classnames libraries
 */
export function cn(...classes: (string | undefined | null | false)[]): string {
  return classes.filter(Boolean).join(' ');
}
