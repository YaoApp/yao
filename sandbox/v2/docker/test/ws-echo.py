"""WebSocket echo server on port 9800 using the websockets library."""
import asyncio
import websockets

async def echo(ws):
    async for msg in ws:
        await ws.send(msg)

async def main():
    async with websockets.serve(echo, "0.0.0.0", 9800):
        await asyncio.Future()

if __name__ == "__main__":
    asyncio.run(main())
