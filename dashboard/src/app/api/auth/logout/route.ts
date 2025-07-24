import { NextResponse } from 'next/server';

export async function GET(request: Request) {
  try {
    const response = await fetch('http://localhost:8080/auth/logout', {
      headers: {
        'Cookie': request.headers.get('Cookie') || '',
      },
    });

    if (response.ok) {
      return NextResponse.json({ message: 'Logged out' }, { status: response.status });
    } else {
      return NextResponse.json({ message: 'Logout failed' }, { status: response.status });
    }
  } catch (error) {
    console.error('Error during logout:', error);
    return NextResponse.json({ message: 'Internal Server Error' }, { status: 500 });
  }
}
