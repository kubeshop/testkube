using System.Threading.Tasks;
using Xunit;

public class ExampleTests
{
    [Fact]
    public async Task Test1_ShouldBeTrue()
    {
        await Task.Delay(500);
        Assert.True(2 + 2 == 4);
    }

    [Fact]
    public async Task Test2_ShouldBeFalse()
    {
        await Task.Delay(1000);
        Assert.False(5 < 3);
    }

    [Fact]
    public void Test3_ShouldEqual()
    {
        Assert.Equal("abc", "a" + "b" + "c");
    }

    [Fact]
    public async Task Test4_ShouldNotEqual()
    {
        await Task.Delay(3000);
        Assert.NotEqual(10, 5);
    }

    [Fact]
    public void Test5_ShouldBeNull()
    {
        string value = null;
        Assert.Null(value);
    }
}