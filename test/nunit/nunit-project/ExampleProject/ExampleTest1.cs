namespace ExampleProject;

public class Tests
{
    [SetUp]
    public void Setup()
    {
    }

    [Test]
    public void Test1()
    {
        Assert.Pass();
    }

    [Test]
    public void Test2()
    {
        System.Threading.Thread.Sleep(4000);
        Assert.Pass();
    }

    [Test]
    public void Test3()
    {
        System.Threading.Thread.Sleep(500);
        Assert.Pass();
    }

    [Test]
    public void Test4()
    {
        System.Threading.Thread.Sleep(1000);
        Assert.Pass();
    }

    [Test]
    public void Test5()
    {
        System.Threading.Thread.Sleep(6000);
        Assert.Pass();
    }

    [Test]
    public void TestA() {
        System.Threading.Thread.Sleep(2000);
        var variable1 = "A";
        var variable2 = "A";
        Assert.That(variable1, Is.EqualTo(variable2));
    }

    [Test]
    public void TestB() {
        System.Threading.Thread.Sleep(500);
        var variable1 = "B";
        var variable2 = "B";
        Assert.That(variable1, Is.EqualTo(variable2));
    }

    [Test]
    public void TestC() {
        System.Threading.Thread.Sleep(1000);
        var variable1 = "C";
        var variable2 = "C";
        Assert.That(variable1, Is.EqualTo(variable2));
    }
}
