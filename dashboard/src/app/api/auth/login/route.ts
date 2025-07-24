import { NextResponse } from 'next/server';

export async function GET(request: Request) {
  try {
    const response = await fetch('http://localhost:8080/auth/login', {
      headers: {
        'Cookie': request.headers.get('Cookie') || '',
      },
    });

    if (response.ok) {
      const data = await response.json();
      // GoバックエンドからのリダイレクトURLをそのまま返す
      return NextResponse.json(data, { status: response.status });
    } else {
      return NextResponse.json({ message: 'Login failed' }, { status: response.status });
    }
  } catch (error) {
    console.error('Error during login:', error);
    return NextResponse.json({ message: 'Internal Server Error' }, { status: 500 });
  }
}
