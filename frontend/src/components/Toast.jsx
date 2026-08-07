import { useToast } from '../hooks/useToast';
import { X, CheckCircle, AlertCircle, Info } from 'lucide-react';
import { clsx } from 'clsx';
import { motion, AnimatePresence } from 'framer-motion';

export default function Toast() {
  const { toasts, removeToast } = useToast();

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-3 w-full max-w-sm">
      <AnimatePresence>
        {toasts.map((toast) => (
          <motion.div
            key={toast.id}
            layout
            initial={{ opacity: 0, x: 400, scale: 0.8 }}
            animate={{ opacity: 1, x: 0, scale: 1 }}
            exit={{ opacity: 0, x: 400, scale: 0.8 }}
            transition={{ type: 'spring', stiffness: 350, damping: 30 }}
            className={clsx(
              "glass flex items-center p-4 rounded-2xl border shadow-glass dark:shadow-glass-dark relative overflow-hidden",
              toast.type === 'success' && "border-green-400/40 text-green-800 dark:text-green-200",
              toast.type === 'error' && "border-red-400/40 text-red-800 dark:text-red-200",
              toast.type === 'info' && "border-blue-400/40 text-blue-800 dark:text-blue-200"
            )}
          >
            <motion.div
              initial={{ scale: 0, rotate: -180 }}
              animate={{ scale: 1, rotate: 0 }}
              transition={{ type: 'spring', stiffness: 400, damping: 15, delay: 0.1 }}
              className="flex-shrink-0 mr-3"
            >
              {toast.type === 'success' && <CheckCircle className="w-5 h-5 text-green-500 drop-shadow-[0_0_8px_rgba(16,185,129,0.6)]" />}
              {toast.type === 'error' && <AlertCircle className="w-5 h-5 text-red-500 drop-shadow-[0_0_8px_rgba(239,68,68,0.6)]" />}
              {toast.type === 'info' && <Info className="w-5 h-5 text-blue-500 drop-shadow-[0_0_8px_rgba(59,130,246,0.6)]" />}
            </motion.div>
            <p className="text-sm font-medium flex-grow">{toast.message}</p>
            <button
              onClick={() => removeToast(toast.id)}
              className="flex-shrink-0 ml-3 p-1 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-white/30 dark:hover:bg-white/10 transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
            {/* Auto-dismiss progress bar */}
            <motion.div
              className={clsx(
                "absolute bottom-0 left-0 h-1",
                toast.type === 'success' && "bg-green-500",
                toast.type === 'error' && "bg-red-500",
                toast.type === 'info' && "bg-blue-500"
              )}
              initial={{ width: '100%' }}
              animate={{ width: '0%' }}
              transition={{ duration: 4, ease: 'linear' }}
            />
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}
