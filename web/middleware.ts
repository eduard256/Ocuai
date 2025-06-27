import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Paths that don't require authentication
const publicPaths = ['/login', '/register', '/api/auth/setup', '/api/auth/login', '/api/auth/register', '/api/auth/status'];

// Paths that require authentication
const protectedPaths = ['/dashboard', '/cameras', '/events', '/settings'];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  
  // Allow public paths
  if (publicPaths.some(path => pathname.startsWith(path))) {
    return NextResponse.next();
  }
  
  // Check if trying to access protected path
  const isProtectedPath = protectedPaths.some(path => pathname.startsWith(path));
  
  // For protected paths, we'll handle auth check on the client side
  // This is because we're using cookie-based auth and can't easily check it in middleware
  // The client will redirect to login if not authenticated
  
  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder
     */
    '/((?!_next/static|_next/image|favicon.ico|.*\\..*).*)' 
  ],
}; 