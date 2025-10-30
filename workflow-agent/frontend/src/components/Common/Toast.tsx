import { useEffect } from 'react';
import { CheckCircle2, XCircle, AlertCircle, Info, X } from 'lucide-react';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

interface ToastProps {
  message: string;
  type?: ToastType;
  duration?: number;
  onClose: () => void;
}

export function Toast({ message, type = 'info', duration = 3000, onClose }: ToastProps) {
  useEffect(() => {
    if (duration > 0) {
      const timer = setTimeout(onClose, duration);
      return () => clearTimeout(timer);
    }
  }, [duration, onClose]);

  const getIcon = () => {
    switch (type) {
      case 'success':
        return <CheckCircle2 className="h-5 w-5" />;
      case 'error':
        return <XCircle className="h-5 w-5" />;
      case 'warning':
        return <AlertCircle className="h-5 w-5" />;
      case 'info':
        return <Info className="h-5 w-5" />;
    }
  };

  const getGradient = () => {
    switch (type) {
      case 'success':
        return 'from-emerald-500 to-cyan-500';
      case 'error':
        return 'from-rose-500 to-fuchsia-500';
      case 'warning':
        return 'from-amber-500 to-rose-500';
      case 'info':
        return 'from-violet-500 to-fuchsia-500';
    }
  };

  const getTextColor = () => {
    switch (type) {
      case 'success':
        return 'text-emerald-600 dark:text-emerald-400';
      case 'error':
        return 'text-rose-600 dark:text-rose-400';
      case 'warning':
        return 'text-amber-600 dark:text-amber-400';
      case 'info':
        return 'text-violet-600 dark:text-violet-400';
    }
  };

  return (
    <div className="relative group animate-slide-in-right pointer-events-auto">
      {/* Gradient glow background */}
      <div className={`absolute inset-0 rounded-2xl bg-gradient-to-r ${getGradient()} opacity-20 blur-xl`}></div>

      {/* Glass container */}
      <div className="relative glass-strong rounded-2xl shadow-[0_8px_32px_-4px_rgba(0,0,0,0.2)] min-w-[320px] max-w-md">
        <div className="flex items-start gap-3 p-4">
          {/* Icon with gradient */}
          <div className="relative flex-shrink-0">
            <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${getGradient()} flex items-center justify-center shadow-lg`}>
              {getIcon()}
              <div className="absolute inset-0 text-white">{getIcon()}</div>
            </div>
            {/* Icon glow */}
            <div className={`absolute inset-0 rounded-xl blur-md opacity-50 bg-gradient-to-br ${getGradient()}`}></div>
          </div>

          {/* Message */}
          <p className={`flex-1 text-sm font-medium leading-relaxed pt-2 ${getTextColor()}`}>
            {message}
          </p>

          {/* Close button */}
          <button
            onClick={onClose}
            className="flex-shrink-0 w-8 h-8 rounded-lg hover:bg-slate-500/10 dark:hover:bg-slate-400/10 transition-colors flex items-center justify-center group/close"
            aria-label="关闭"
          >
            <X className="h-4 w-4 text-slate-500 dark:text-slate-400 group-hover/close:text-slate-700 dark:group-hover/close:text-slate-300 transition-colors" />
          </button>
        </div>

        {/* Progress bar */}
        {duration > 0 && (
          <div className="h-1 bg-slate-200/50 dark:bg-slate-700/50 rounded-b-2xl overflow-hidden">
            <div
              className={`h-full bg-gradient-to-r ${getGradient()}`}
              style={{
                animation: `shrink ${duration}ms linear forwards`,
              }}
            ></div>
          </div>
        )}
      </div>

      <style>{`
        @keyframes shrink {
          from {
            width: 100%;
          }
          to {
            width: 0%;
          }
        }
      `}</style>
    </div>
  );
}

// Toast Container Component
interface ToastContainerProps {
  toasts: Array<{
    id: string;
    message: string;
    type: ToastType;
    duration?: number;
  }>;
  onRemove: (id: string) => void;
}

export function ToastContainer({ toasts, onRemove }: ToastContainerProps) {
  return (
    <div className="fixed top-20 right-6 z-[100] flex flex-col gap-3 pointer-events-none">
      {toasts.map((toast) => (
        <Toast
          key={toast.id}
          message={toast.message}
          type={toast.type}
          duration={toast.duration}
          onClose={() => onRemove(toast.id)}
        />
      ))}
    </div>
  );
}
