import { Github, Twitter, Heart } from 'lucide-react';

export default function Footer() {
  return (
    <footer className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 py-8 mt-auto">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row justify-between items-center space-y-4 md:space-y-0">
          <div className="text-gray-600 dark:text-gray-400 text-sm flex items-center">
            Built with <Heart className="w-4 h-4 mx-1 text-red-500 fill-current" /> by LinkSnip Enterprise
          </div>
          <div className="flex space-x-6">
            <a href="#" className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors">
              <Github className="w-5 h-5" />
            </a>
            <a href="#" className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors">
              <Twitter className="w-5 h-5" />
            </a>
          </div>
          <div className="text-gray-500 dark:text-gray-400 text-sm">
            &copy; {new Date().getFullYear()} LinkSnip. All rights reserved.
          </div>
        </div>
      </div>
    </footer>
  );
}
